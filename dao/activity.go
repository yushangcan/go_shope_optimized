package dao

import "go_shope/model"

func (r *Repository) CreateActivity(activity *model.SeckillActivity) error {
	return r.DB.Create(activity).Error
}

func (r *Repository) ListActivities() ([]model.SeckillActivity, error) {
	var activities []model.SeckillActivity
	if err := r.DB.Preload("Product").Order("start_time desc").Find(&activities).Error; err != nil {
		return nil, err
	}
	return activities, nil
}

func (r *Repository) FindActivityByID(id uint64) (*model.SeckillActivity, error) {
	var activity model.SeckillActivity
	if err := r.DB.Preload("Product").First(&activity, id).Error; err != nil {
		return nil, err
	}
	return &activity, nil
}

func (r *Repository) UpdateActivity(activity *model.SeckillActivity) error {
	return r.DB.Save(activity).Error
}
func (r *Repository) DeleteActivity(id uint64) error {
	return r.DB.Delete(&model.SeckillActivity{}, id).Error
}
