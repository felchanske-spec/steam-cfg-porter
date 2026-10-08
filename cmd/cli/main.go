package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	pb "steam-cfg-portet/api/pb"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/caarlos0/env/v11"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type screenState int

const (
	screenHome screenState = iota
	screenChooseGame
	screenDotaConfigs
	screenCS2Configs
	screenMakeConfig
	screenMyConfigs
	screenMyConfigAction
)

type gRPCSuccessMsg string
type gRPCErrorMsg string

type gRPCListResponseMsg struct {
	configs []*pb.ConfigModel
}

type Config struct {
	ConfigServiceGRPCAddr string `env:"CONFIG_SERVICE_GRPC_ADDR" envDefault:"localhost:50051"`
}

type model struct {
	state          screenState
	cursor         int
	menuItems      []string
	selectedConfig string
	statusMessage  string
	isSuccess      bool
	windowStyle    lipgloss.Style
	client         pb.ConfigServiceClient
	targetCfgDir   string
}

func fetchConfigsListCmd(client pb.ConfigServiceClient, game string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		resp, err := client.ListConfig(ctx, &pb.ListConfigsRequest{Game: game})
		if err != nil {
			return gRPCErrorMsg(fmt.Sprintf("Failed to load list: %v", grpc.ErrorDesc(err)))
		}

		return gRPCListResponseMsg{configs: resp.GetConfigs()}
	}
}

func initialModel(client pb.ConfigServiceClient) model {
	return model{
		state:        screenHome,
		cursor:       0,
		menuItems:    []string{"Select config", "Save config", "My saves", "Exit"},
		client:       client,
		targetCfgDir: "./tmp/steam_configs",
		windowStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2).
			Margin(1, 1),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case gRPCListResponseMsg:
		m.statusMessage = ""
		if len(msg.configs) == 0 {
			m.menuItems = []string{"No configs available in cloud"}
			m.cursor = 0
			return m, nil
		}

		m.menuItems = make([]string, 0, len(msg.configs))
		for _, cfg := range msg.configs {
			itemStr := fmt.Sprintf("%s (Author: %s) [ID: %s]", cfg.GetName(), cfg.GetAuthorNickname(), cfg.GetId())
			m.menuItems = append(m.menuItems, itemStr)
		}
		m.cursor = 0
		return m, nil
	case gRPCSuccessMsg:
		m.statusMessage = string(msg)
		m.isSuccess = true
		return m, nil

	case gRPCErrorMsg:
		m.statusMessage = string(msg)
		m.isSuccess = false
		return m, nil

	case tea.KeyPressMsg:
		m.statusMessage = ""

		switch msg.Key().Code {
		case tea.KeyEnter:
			return m.handleSelect()
		case tea.KeyEsc, tea.KeyBackspace:
			return m.handleBack()
		}

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.menuItems)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m model) handleBack() (tea.Model, tea.Cmd) {
	switch m.state {
	case screenChooseGame:
		m.state = screenHome
		m.menuItems = []string{"Select config", "Save config", "My saves", "Exit"}
	case screenDotaConfigs, screenCS2Configs:
		m.state = screenChooseGame
		m.menuItems = []string{"Dota 2", "CS 2"}
	case screenMakeConfig, screenMyConfigs:
		m.state = screenHome
		m.menuItems = []string{"Select config", "Save config", "My saves", "Exit"}
	case screenMyConfigAction:
		m.state = screenMyConfigs
		m.menuItems = []string{"dota2_stariy_bog_config", "dota2_yatoro_config", "cs2_s1mple_config", "cs2_donk_config"}
	}
	m.cursor = 0
	return m, nil
}

func (m model) handleSelect() (tea.Model, tea.Cmd) {
	if m.client == nil {
		m.statusMessage = "gRPC Error: Client connection is not initialized"
		m.isSuccess = false
		return m, nil
	}

	switch m.state {
	case screenHome:
		switch m.cursor {
		case 0:
			m.state = screenChooseGame
			m.menuItems = []string{"Dota 2", "CS 2"}
		case 1:
			m.state = screenMakeConfig
			m.menuItems = []string{"Press Enter to Save", "Back"}
		case 2:
			m.state = screenMyConfigs
			m.menuItems = []string{"dota2_stariy_bog_config", "dota2_yatoro_config", "cs2_s1mple_config", "cs2_donk_config"}
		case 3:
			return m, tea.Quit
		}
		m.cursor = 0

	case screenChooseGame:
		selectedGame := m.menuItems[m.cursor]
		switch selectedGame {
		case "Dota 2":
			m.state = screenDotaConfigs
			m.targetCfgDir = detectSteamPath("dota2")
			m.statusMessage = "Loading configs from cloud..."
			return m, fetchConfigsListCmd(m.client, "dota2")
		case "CS 2":
			m.state = screenCS2Configs
			m.targetCfgDir = detectSteamPath("cs2")
			m.statusMessage = "Loading configs from cloud..."
			return m, fetchConfigsListCmd(m.client, "cs2")
		}
		m.cursor = 0

	case screenDotaConfigs, screenCS2Configs:
		if len(m.menuItems) == 0 || m.menuItems[0] == "No configs available in cloud" {
			return m, nil
		}

		selectedItem := m.menuItems[m.cursor]

		idStart := strings.Index(selectedItem, "[")
		idEnd := strings.Index(selectedItem, "]")
		if idStart == -1 || idEnd == -1 {
			return m, nil
		}
		configID := selectedItem[idStart+1 : idEnd]

		gameType := "dota2"
		if m.state == screenCS2Configs {
			gameType = "cs2"
		}

		m.statusMessage = "Installing configuration..."

		return m, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			_, err := m.client.GetConfig(ctx, &pb.GetConfigRequest{
				ConfigId: configID,
				Game:     gameType,
			})
			if err != nil {
				return gRPCErrorMsg(fmt.Sprintf("Download failed: %v", grpc.ErrorDesc(err)))
			}
			return gRPCSuccessMsg("Config successfully installed!")
		}

	case screenMakeConfig:
		if m.cursor == 1 {
			return m.handleBack()
		}
		return m, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_, err := m.client.SaveConfig(ctx, &pb.SaveConfigRequest{
				Game:           "dota2",
				Name:           "custom_preset",
				AuthorNickname: "leha",
				UserId:         12345,
			})
			if err != nil {
				return gRPCErrorMsg(fmt.Sprintf("Save failed: %v", grpc.ErrorDesc(err)))
			}
			return gRPCSuccessMsg("Config saved successfully to cloud!")
		}

	case screenMyConfigs:
		m.selectedConfig = m.menuItems[m.cursor]
		m.state = screenMyConfigAction
		m.menuItems = []string{"Choose (Install)", "Delete From Cloud"}
		m.cursor = 0

	case screenMyConfigAction:
		gameType := "dota2"
		if len(m.selectedConfig) >= 3 && m.selectedConfig[:3] == "cs2" {
			gameType = "cs2"
		}
		if m.cursor == 0 {
			return m, func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				_, err := m.client.GetConfig(ctx, &pb.GetConfigRequest{
					ConfigId: m.selectedConfig,
					Game:     gameType,
				})
				if err != nil {
					return gRPCErrorMsg(fmt.Sprintf("Install failed: %v", grpc.ErrorDesc(err)))
				}
				return gRPCSuccessMsg("Your config installed successfully!")
			}
		} else {
			return m, func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				_, err := m.client.DeleteConfig(ctx, &pb.DeleteConfigRequest{
					ConfigId: m.selectedConfig,
					UserId:   12345,
				})
				if err != nil {
					return gRPCErrorMsg(fmt.Sprintf("Delete failed: %v", grpc.ErrorDesc(err)))
				}
				return gRPCSuccessMsg("Config deleted from cloud!")
			}
		}
	}
	return m, nil
}

func (m model) View() tea.View {
	var s string
	switch m.state {
	case screenHome:
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Render("registry://home\n\n")
		s += "CONFIG REGISTRY\n\n"
		s += m.renderMenu()
	case screenChooseGame:
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Render("select://game\n\n")
		s += "CHOOSE GAME      STEP 01\n\n"
		s += m.renderMenu()
	case screenDotaConfigs:
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Render("select://dota2\n\n")
		s += "DOTA 2 CONFIGS   STEP 02\n\n"
		s += m.renderMenu()
	case screenCS2Configs:
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Render("select://cs2\n\n")
		s += "CS 2 CONFIGS     STEP 02\n\n"
		s += m.renderMenu()
	case screenMakeConfig:
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("76")).Render("create://new\n\n")
		s += "MAKE A CONFIG PRESET\n\n"
		s += "🔹 Game: dota2\n"
		s += "🔹 Name: custom_preset\n"
		s += "🔹 Author: leha\n\n"
		s += m.renderMenu()
	case screenMyConfigs:
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("135")).Render("registry://saves\n\n")
		s += "MY CONFIGS (SAVES)\n\n"
		s += m.renderMenu()
	case screenMyConfigAction:
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("135")).Render("registry://saves/action\n\n")
		s += fmt.Sprintf("TARGET: %s\n\n", lipgloss.NewStyle().Bold(true).Render(m.selectedConfig))
		s += m.renderMenu()
	}

	if m.statusMessage != "" {
		s += "\n"
		if m.isSuccess {
			s += lipgloss.NewStyle().Foreground(lipgloss.Color("76")).Bold(true).Render(m.statusMessage)
		} else {
			s += lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Render(m.statusMessage)
		}
	}

	if m.state != screenHome {
		s += "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("(Esc: Back | Ctrl+C: Quit)")
	} else {
		s += "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("(Ctrl+C: Quit)")
	}

	return tea.NewView(m.windowStyle.Render(s))
}

func (m model) renderMenu() string {
	var s string
	for i, item := range m.menuItems {
		if m.cursor == i {
			s += fmt.Sprintf("%s\n", lipgloss.NewStyle().
				Foreground(lipgloss.Color("33")).
				Bold(true).
				Render(fmt.Sprintf("➡️  %s", item)))
		} else {
			s += fmt.Sprintf("    %s\n", item)
		}
	}
	return s
}

func detectSteamPath(game string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./tmp/steam_configs"
	}

	var basePath string

	switch runtime.GOOS {
	case "darwin":
		basePath = filepath.Join(homeDir, "Library", "Application Support", "Steam", "steamapps", "common")
	case "windows":
		basePath = "C:\\Program Files (x86)\\Steam\\steamapps\\common"
	case "linux":
		basePath = filepath.Join(homeDir, ".local", "share", "Steam", "steamapps", "common")
	default:
		return "./tmp/steam_configs"
	}

	var gameSubPath string
	switch game {
	case "dota2":
		gameSubPath = filepath.Join("dota 2 beta", "game", "dota", "cfg")
	case "cs2":
		gameSubPath = filepath.Join("Counter-Strike Global Offensive", "game", "csgo", "cfg")
	default:
		return "./tmp/steam_configs"
	}

	fullPath := filepath.Join(basePath, gameSubPath)

	if _, err := os.Stat(fullPath); err == nil {
		return fullPath
	}

	return ""
}

func main() {
	var envCfg Config
	if err := env.Parse(&envCfg); err != nil {
		log.Fatalf("Failed to parse env config: %v", err)
	}

	conn, err := grpc.NewClient(envCfg.ConfigServiceGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("gRPC connection initialization failed: %v", err)
	}
	defer conn.Close()

	gRPCClient := pb.NewConfigServiceClient(conn)
	p := tea.NewProgram(initialModel(gRPCClient))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
