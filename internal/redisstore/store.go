package redisstore

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrUnavailable = errors.New("redis unavailable")

const admissionScript = `
local req = KEYS[1]
local buyer = KEYS[2]
local activity = KEYS[3]
local request_id = ARGV[1]
local user_id = ARGV[2]
local activity_id = ARGV[3]
local now = tonumber(ARGV[4])
local start_at = tonumber(redis.call('HGET', activity, 'start_at') or '0')
local end_at = tonumber(redis.call('HGET', activity, 'end_at') or '0')
local status = redis.call('HGET', activity, 'status') or ''
local current = redis.call('HGET', req, 'status')
if current then
  local owner = redis.call('HGET', req, 'user_id')
  if owner and owner ~= user_id then return {5, 'REQUEST_ID_CONFLICT'} end
  return {1, current}
end
if status ~= 'ACTIVE' or now < start_at or now >= end_at then return {4, 'ACTIVITY_NOT_ACTIVE'} end
if redis.call('SISMEMBER', buyer, user_id) == 1 then return {3, 'DUPLICATE_BUYER'} end
local stock = tonumber(redis.call('HGET', activity, 'stock') or '0')
if stock <= 0 then return {2, 'SOLD_OUT'} end
redis.call('HINCRBY', activity, 'stock', -1)
redis.call('SADD', buyer, user_id)
redis.call('HSET', req, 'status', 'ACCEPTED', 'user_id', user_id, 'activity_id', activity_id, 'updated_at', now)
redis.call('EXPIRE', req, 86400)
return {0, 'ACCEPTED'}`

type OrderEvent struct {
	RequestID  string `json:"request_id"`
	UserID     uint64 `json:"user_id"`
	ActivityID uint64 `json:"activity_id"`
}

type Store struct {
	Client *redis.Client
}

func New(addr, password string, db int) *Store {
	return &Store{Client: redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db, DialTimeout: 2 * time.Second, ReadTimeout: 2 * time.Second, WriteTimeout: 2 * time.Second, PoolSize: 32})}
}

func (s *Store) Ping(ctx context.Context) error {
	if err := s.Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return nil
}

func (s *Store) PublishActivity(ctx context.Context, activityID uint64, status string, start, end time.Time, stock int) error {
	key := fmt.Sprintf("seckill:activity:%d", activityID)
	return s.Client.HSet(ctx, key, map[string]any{"status": status, "start_at": start.Unix(), "end_at": end.Unix(), "stock": stock}).Err()
}

func (s *Store) Admit(ctx context.Context, activityID, userID uint64, requestID string, now time.Time) (int, string, error) {
	result, err := s.Client.Eval(ctx, admissionScript, []string{fmt.Sprintf("seckill:request:%s", requestID), fmt.Sprintf("seckill:buyers:%d", activityID), fmt.Sprintf("seckill:activity:%d", activityID)}, requestID, strconv.FormatUint(userID, 10), strconv.FormatUint(activityID, 10), now.Unix()).Result()
	if err != nil {
		return -1, "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	values, ok := result.([]any)
	if !ok || len(values) != 2 {
		return -1, "", errors.New("invalid redis admission response")
	}
	code, _ := values[0].(int64)
	status, _ := values[1].(string)
	return int(code), status, nil
}

func (s *Store) RequestStatus(ctx context.Context, requestID string) (map[string]string, error) {
	if strings.TrimSpace(requestID) == "" {
		return nil, errors.New("request_id is required")
	}
	return s.Client.HGetAll(ctx, fmt.Sprintf("seckill:request:%s", requestID)).Result()
}

func (s *Store) Mark(ctx context.Context, requestID, status string, fields map[string]any) error {
	if fields == nil {
		fields = make(map[string]any)
	}
	fields["status"] = status
	fields["updated_at"] = time.Now().Unix()
	return s.Client.HSet(ctx, fmt.Sprintf("seckill:request:%s", requestID), fields).Err()
}
func (s *Store) Compensate(ctx context.Context, event OrderEvent, reason string) error {
	key := fmt.Sprintf("seckill:request:%s", event.RequestID)
	activity := fmt.Sprintf("seckill:activity:%d", event.ActivityID)
	buyer := fmt.Sprintf("seckill:buyers:%d", event.ActivityID)
	script := `local current=redis.call('HGET',KEYS[1],'status'); if current=='SUCCEEDED' or current=='FAILED' then return 0 end; redis.call('HINCRBY',KEYS[2],'stock',1); redis.call('SREM',KEYS[3],ARGV[1]); redis.call('HSET',KEYS[1],'status','FAILED','reason',ARGV[2]); return 1`
	return s.Client.Eval(ctx, script, []string{key, activity, buyer}, strconv.FormatUint(event.UserID, 10), reason).Err()
}
