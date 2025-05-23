package models

import "github.com/ZaparooProject/zaparoo-core/pkg/zapscript/models"

type NoticeArgs struct {
	Text     string `json:"text"`
	Timeout  int    `json:"timeout"`
	Complete string `json:"complete"`
}

type PickerArgs struct {
	Items    []models.ZapScript `json:"items"`
	Title    string             `json:"title"`
	Selected int                `json:"selected"`
	Timeout  int                `json:"timeout"`
	Unsafe   bool               `json:"unsafe"`
}
