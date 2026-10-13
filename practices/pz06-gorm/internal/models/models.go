package models

import "time"

type User struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"not null"`
	Email     string `gorm:"not null;uniqueIndex"`
	Notes     []Note `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Note struct {
	ID        uint   `gorm:"primaryKey"`
	Title     string `gorm:"not null"`
	Content   string `gorm:"not null"`
	UserID    uint   `gorm:"not null;index"`
	User      User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Tags      []Tag  `gorm:"many2many:note_tags;"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Tag struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"not null;uniqueIndex"`
	Notes     []Note `gorm:"many2many:note_tags;"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
