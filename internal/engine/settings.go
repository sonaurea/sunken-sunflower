package engine

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

type Resolution struct {
	Name   string
	Width  int
	Height int
}

var AvailableResolutions = []Resolution{
	{Name: "1280x720 (HD)", Width: 1280, Height: 720},
	{Name: "1366x768", Width: 1366, Height: 768},
	{Name: "1600x900", Width: 1600, Height: 900},
	{Name: "1920x1080 (FHD)", Width: 1920, Height: 1080},
	{Name: "2560x1440 (QHD)", Width: 2560, Height: 1440},
	{Name: "3840x2160 (4K)", Width: 3840, Height: 2160},
}

type DisplayConfig struct {
	ResolutionIndex int  `json:"res"`
	VSync           bool `json:"vsync"`
	FPSCap          int  `json:"fps"`
	ShowFPSCounter  bool `json:"show_fps"`
	Fullscreen      bool `json:"fullscreen"`
}

func DefaultDisplayConfig() DisplayConfig {
	return DisplayConfig{ResolutionIndex: 0, VSync: true, FPSCap: 60}
}

func (dc *DisplayConfig) Resolution() Resolution {
	return AvailableResolutions[dc.ResolutionIndex]
}

type SettingsMenu struct {
	Active   bool
	Config   DisplayConfig
	options  []string
	selected int
}

func NewSettingsMenu() *SettingsMenu {
	return &SettingsMenu{
		Config: DefaultDisplayConfig(),
		options: []string{"Resolution", "Fullscreen", "VSync", "FPS Cap", "Show FPS"},
	}
}

func (sm *SettingsMenu) Update(input *Input) {
	if !sm.Active {
		return
	}
	if input.UpJustPressed {
		sm.selected--
		if sm.selected < 0 {
			sm.selected = len(sm.options) - 1
		}
	}
	if input.DownJustPressed {
		sm.selected++
		if sm.selected >= len(sm.options) {
			sm.selected = 0
		}
	}
	if input.LeftJustPressed || input.RightJustPressed {
		switch sm.selected {
		case 0:
			sm.Config.ResolutionIndex += boolToDir(input.RightJustPressed) - boolToDir(input.LeftJustPressed)
			if sm.Config.ResolutionIndex < 0 {
				sm.Config.ResolutionIndex = len(AvailableResolutions) - 1
			}
			if sm.Config.ResolutionIndex >= len(AvailableResolutions) {
				sm.Config.ResolutionIndex = 0
			}
		case 1:
			sm.Config.Fullscreen = !sm.Config.Fullscreen
		case 2:
			sm.Config.VSync = !sm.Config.VSync
		case 3:
			fpsOpts := []int{30, 60, 120, 144, 0}
			for i, f := range fpsOpts {
				if f == sm.Config.FPSCap {
					sm.Config.FPSCap = fpsOpts[(i+1)%len(fpsOpts)]
					break
				}
			}
		case 4:
			sm.Config.ShowFPSCounter = !sm.Config.ShowFPSCounter
		}
	}
}

func (sm *SettingsMenu) Draw(screen *ebiten.Image) {
	if !sm.Active {
		return
	}
	DrawRect(screen, 0, 0, float64(ScreenWidth), float64(ScreenHeight), color.RGBA{0, 0, 0, 200})
	title := "[Set] Settings"
	DrawText(screen, title, CenterX(title, ScreenWidth), 60, ColSunflower)
	startY := 120
	values := []string{
		AvailableResolutions[sm.Config.ResolutionIndex].Name,
		boolStr(sm.Config.Fullscreen),
		boolStr(sm.Config.VSync),
		fpsStr(sm.Config.FPSCap),
		boolStr(sm.Config.ShowFPSCounter),
	}
	for i, opt := range sm.options {
		y := startY + i*40
		m := "  "
		clr := ColWhite
		if i == sm.selected {
			m = "-> "
			clr = ColSunflower
		}
		DrawText(screen, m+opt, 300, y, clr)
		DrawText(screen, values[i], 550, y, ColBeachPink)
	}
	note := "Isometric view scales to all resolutions"
	DrawText(screen, note, CenterX(note, ScreenWidth), ScreenHeight-100, color.RGBA{150, 150, 150, 255})
	DrawText(screen, "[ESC] Close", ScreenWidth-120, ScreenHeight-30, ColWhite)
}

func (sm *SettingsMenu) Toggle() { sm.Active = !sm.Active }
func boolStr(b bool) string {
	if b {
		return "ON"
	}
	return "OFF"
}
func fpsStr(f int) string {
	if f <= 0 {
		return "Unlimited"
	}
	return fmt.Sprintf("%d FPS", f)
}
func boolToDir(b bool) int {
	if b {
		return 1
	}
	return 0
}