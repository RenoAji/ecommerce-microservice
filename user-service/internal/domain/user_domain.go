package domain

import (
	"time"
)

type User struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"username" binding:"required"`
	Email     string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"email" binding:"required,email"`
	Password  string    `gorm:"type:varchar(255);not null" json:"password" binding:"required,min=6"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	Role      string    `gorm:"type:varchar(20);not null;default:'user'" json:"role" binding:"omitempty,oneof=admin user"`
}
