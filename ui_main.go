package main

import (
	"fmt"
	"sort"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// appState holds the result of a background data fetch.
type appState struct {
	router     *KeeneticRouter
	routerInfo *RouterConfig
	device     *Client
	policies   map[string]interface{}
	err        string
}

// MainUI is the root screen of the mobile app.
type MainUI struct {
	app    fyne.App
	window fyne.Window

	mu      sync.Mutex
	routers []RouterConfig
	state   *appState

	// dynamic widgets
	statusCard  *widget.Card
	deviceCard  *widget.Card
	policyGroup *widget.RadioGroup
	applyBtn    *widget.Button
	refreshBtn  *widget.Button
	spinner     *widget.ProgressBarInfinite
}

func newMainUI(a fyne.App, w fyne.Window) *MainUI {
	ui := &MainUI{
		app:     a,
		window:  w,
		routers: loadRouters(),
	}
	return ui
}

func (ui *MainUI) content() fyne.CanvasObject {
	// ── Toolbar ──────────────────────────────────────────────
	settingsBtn := widget.NewButtonWithIcon("Settings", theme.SettingsIcon(), func() {
		ui.openSettings()
	})
	toolbar := container.NewBorder(nil, nil, nil, settingsBtn,
		widget.NewLabelWithStyle("Keenetic Tray", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)

	// ── Status card ───────────────────────────────────────────
	ui.statusCard = widget.NewCard("Router", "Searching...", nil)

	// ── Device card ───────────────────────────────────────────
	ui.deviceCard = widget.NewCard("This Device", "", nil)

	// ── Policy selector ───────────────────────────────────────
	ui.policyGroup = widget.NewRadioGroup(nil, nil)

	ui.applyBtn = widget.NewButton("Apply Policy", ui.onApply)
	ui.applyBtn.Importance = widget.HighImportance
	ui.applyBtn.Disable()

	// ── Refresh ───────────────────────────────────────────────
	ui.spinner = widget.NewProgressBarInfinite()
	ui.spinner.Hide()

	ui.refreshBtn = widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		ui.refresh()
	})

	policySection := widget.NewCard("Access Policy", "", container.NewVBox(
		ui.policyGroup,
		ui.applyBtn,
	))

	scroll := container.NewVScroll(container.NewVBox(
		ui.statusCard,
		ui.deviceCard,
		policySection,
	))

	bottom := container.NewVBox(ui.spinner, ui.refreshBtn)
	root := container.NewBorder(toolbar, bottom, nil, nil, scroll)

	// Initial data load
	go ui.refresh()

	return root
}

// refresh fetches state from the router in a background goroutine.
func (ui *MainUI) refresh() {
	ui.setLoading(true)

	ui.mu.Lock()
	routers := make([]RouterConfig, len(ui.routers))
	copy(routers, ui.routers)
	ui.mu.Unlock()

	state := ui.collectState(routers)

	// Back on "main" — Fyne is safe to update from any goroutine for these widgets
	ui.applyState(state)
	ui.setLoading(false)
}

func (ui *MainUI) collectState(routers []RouterConfig) *appState {
	if len(routers) == 0 {
		return &appState{err: "No routers configured.\nTap Settings → Add Router."}
	}

	localNets := getLocalNetworks()

	for i := range routers {
		ri := &routers[i]
		password := getPassword(ri)
		if password == "" {
			continue
		}

		addr := ""
		if ri.NetworkIP != "" && isIPInNetworks(ri.NetworkIP, localNets) {
			addr = ri.NetworkIP
		} else {
			host := extractHost(ri.Address)
			if host != "" && isIPInNetworks(host, localNets) {
				addr = ri.Address
			}
		}
		if addr == "" {
			continue
		}

		router := NewKeeneticRouter(addr, ri.Login, password, ri.Name)
		if err := router.Login(); err != nil {
			continue
		}

		policies, _ := router.GetPolicies()
		if policies == nil {
			policies = map[string]interface{}{}
		}
		clients, err := router.GetOnlineClients()
		if err != nil {
			return &appState{err: "Error fetching clients: " + err.Error()}
		}

		device := FindThisDevice(clients)
		return &appState{
			router:     router,
			routerInfo: ri,
			device:     device,
			policies:   policies,
		}
	}

	return &appState{err: "No router found on this network.\nMake sure you are connected to the router's Wi-Fi."}
}

func (ui *MainUI) applyState(state *appState) {
	ui.state = state

	if state.err != "" {
		ui.statusCard.SetSubTitle(state.err)
		ui.statusCard.SetTitle("Router")
		ui.deviceCard.SetTitle("This Device")
		ui.deviceCard.SetSubTitle("—")
		ui.deviceCard.SetContent(nil)
		ui.policyGroup.Options = nil
		ui.policyGroup.Refresh()
		ui.applyBtn.Disable()
		return
	}

	// Router status
	ui.statusCard.SetTitle("Router: " + state.routerInfo.Name)
	ui.statusCard.SetSubTitle(state.routerInfo.Address)

	// Device info
	if state.device == nil {
		ui.deviceCard.SetTitle("This Device")
		ui.deviceCard.SetSubTitle("Not found in router's client list")
		ui.deviceCard.SetContent(nil)
		ui.policyGroup.Options = nil
		ui.policyGroup.Refresh()
		ui.applyBtn.Disable()
		return
	}

	d := state.device
	currentLabel := PolicyLabel(d.Policy, state.policies, d.Deny)

	name := d.Name
	if name == "" {
		name = "Unknown"
	}
	deviceInfo := fmt.Sprintf("Name: %s\nIP: %s\nMAC: %s\nPolicy: %s", name, d.IP, d.MAC, currentLabel)
	ui.deviceCard.SetTitle("This Device")
	ui.deviceCard.SetSubTitle(deviceInfo)
	ui.deviceCard.SetContent(nil)

	// Policy options
	options := []string{"Default", "Blocked"}
	for pName, info := range state.policies {
		label := pName
		if m, ok := info.(map[string]interface{}); ok {
			if desc, ok := m["description"].(string); ok && desc != "" {
				label = desc
			}
		}
		options = append(options, label)
	}
	sort.Strings(options[2:]) // keep Default + Blocked first, sort the rest

	ui.policyGroup.Options = options
	ui.policyGroup.SetSelected(currentLabel)
	ui.policyGroup.Refresh()
	ui.applyBtn.Enable()
}

func (ui *MainUI) onApply() {
	state := ui.state
	if state == nil || state.router == nil || state.device == nil {
		return
	}

	selected := ui.policyGroup.Selected
	if selected == "" {
		return
	}

	// Find policy key from label
	policyKey := ""
	switch selected {
	case "Default":
		policyKey = ""
	case "Blocked":
		policyKey = "__blocked__"
	default:
		for pName, info := range state.policies {
			label := pName
			if m, ok := info.(map[string]interface{}); ok {
				if desc, ok := m["description"].(string); ok && desc != "" {
					label = desc
				}
			}
			if label == selected {
				policyKey = pName
				break
			}
		}
	}

	mac := state.device.MAC
	router := state.router

	ui.applyBtn.Disable()
	ui.setLoading(true)

	go func() {
		var err error
		if policyKey == "__blocked__" {
			err = router.SetClientBlock(mac)
		} else {
			err = router.ApplyPolicy(mac, policyKey)
		}
		if err != nil {
			ui.app.SendNotification(&fyne.Notification{
				Title:   "Keenetic Tray",
				Content: "Failed to apply policy: " + err.Error(),
			})
		}
		ui.refresh()
	}()
}

func (ui *MainUI) setLoading(loading bool) {
	if loading {
		ui.spinner.Show()
		ui.refreshBtn.Disable()
	} else {
		ui.spinner.Hide()
		ui.refreshBtn.Enable()
	}
}

func (ui *MainUI) openSettings() {
	ui.mu.Lock()
	routers := make([]RouterConfig, len(ui.routers))
	copy(routers, ui.routers)
	ui.mu.Unlock()

	showSettingsWindow(ui.app, routers, func(updated []RouterConfig) {
		ui.mu.Lock()
		ui.routers = updated
		ui.mu.Unlock()
		go ui.refresh()
	})
}
