package db

import "time"

type Admin struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Username     string `gorm:"uniqueIndex;size:64" json:"username"`
	PasswordHash string `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Setting struct {
	Key   string `gorm:"primaryKey;size:64" json:"key"`
	Value string `json:"value"`
}

type Inbound struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Remark      string    `gorm:"size:128" json:"remark"`
	Enable      bool      `gorm:"default:true" json:"enable"`
	Listen      string    `gorm:"size:64;default:0.0.0.0" json:"listen"`
	Port        int       `gorm:"uniqueIndex" json:"port"`
	Protocol    string    `gorm:"size:32;default:vless" json:"protocol"`
	Network     string    `gorm:"size:32;default:tcp" json:"network"`
	Security    string    `gorm:"size:32;default:reality" json:"security"`
	Dest        string    `gorm:"size:256" json:"dest"`
	ServerNames string    `gorm:"size:512" json:"serverNames"`
	PrivateKey  string    `gorm:"size:128" json:"-"`
	PublicKey   string    `gorm:"size:128" json:"publicKey"`
	ShortIds    string    `gorm:"size:256" json:"shortIds"`
	Fingerprint string    `gorm:"size:32;default:chrome" json:"fingerprint"`
	SpiderX     string    `gorm:"size:128;default:/" json:"spiderX"`
	WSPath      string    `gorm:"size:128" json:"wsPath"`
	WSHost      string    `gorm:"size:256" json:"wsHost"`
	Sniffing    bool      `gorm:"default:true" json:"sniffing"`
	Clients     []Client  `gorm:"constraint:OnDelete:CASCADE" json:"clients,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Client struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	InboundID  uint      `gorm:"index;not null" json:"inboundId"`
	Inbound    *Inbound  `json:"inbound,omitempty"`
	UUID       string    `gorm:"size:64;not null" json:"uuid"`
	Email      string    `gorm:"uniqueIndex;size:128;not null" json:"email"`
	Name       string    `gorm:"size:128" json:"name"`
	Enable     bool      `gorm:"default:true" json:"enable"`
	ExpiryTime int64     `json:"expiryTime"`
	TotalBytes int64     `json:"totalBytes"`
	Up         int64     `json:"up"`
	Down       int64     `json:"down"`
	SubID      string    `gorm:"uniqueIndex;size:32" json:"subId"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func (c Client) Used() int64 {
	return c.Up + c.Down
}

func (c Client) Exhausted() bool {
	if c.TotalBytes <= 0 {
		return false
	}
	return c.Used() >= c.TotalBytes
}

func (c Client) Expired() bool {
	if c.ExpiryTime <= 0 {
		return false
	}
	return time.Now().UnixMilli() >= c.ExpiryTime
}

func (c Client) ActiveForXray() bool {
	return c.Enable && !c.Expired() && !c.Exhausted()
}
