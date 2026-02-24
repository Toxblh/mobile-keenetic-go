package main

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// showSettingsWindow opens a window to manage configured routers.
func showSettingsWindow(a fyne.App, routers []RouterConfig, onSave func([]RouterConfig)) {
	w := a.NewWindow("Settings")
	w.Resize(fyne.NewSize(400, 600))
	w.CenterOnScreen()

	list := widget.NewList(
		func() int { return len(routers) },
		func() fyne.CanvasObject {
			return container.NewBorder(nil, nil, nil,
				widget.NewLabel(""),
				widget.NewLabel(""),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			row := obj.(*fyne.Container)
			row.Objects[0].(*widget.Label).SetText(routers[id].Name)
			row.Objects[1].(*widget.Label).SetText(routers[id].Address)
		},
	)

	selectedIdx := -1
	list.OnSelected = func(id widget.ListItemID) { selectedIdx = id }
	list.OnUnselected = func(_ widget.ListItemID) { selectedIdx = -1 }

	addBtn := widget.NewButton("Add", func() {
		showRouterForm(a, w, nil, routers, func(cfg RouterConfig) {
			routers = append(routers, cfg)
			list.Refresh()
			onSave(routers)
		})
	})
	addBtn.Importance = widget.HighImportance

	editBtn := widget.NewButton("Edit", func() {
		if selectedIdx < 0 || selectedIdx >= len(routers) {
			return
		}
		idx := selectedIdx
		showRouterForm(a, w, &routers[idx], routers, func(cfg RouterConfig) {
			if routers[idx].Name != cfg.Name {
				deletePassword(&routers[idx])
			}
			routers[idx] = cfg
			list.Refresh()
			onSave(routers)
		})
	})

	deleteBtn := widget.NewButton("Delete", func() {
		if selectedIdx < 0 || selectedIdx >= len(routers) {
			return
		}
		idx := selectedIdx
		name := routers[idx].Name
		dialog.ShowConfirm("Delete", fmt.Sprintf("Delete router '%s'?", name), func(ok bool) {
			if !ok {
				return
			}
			deletePassword(&routers[idx])
			routers = append(routers[:idx], routers[idx+1:]...)
			selectedIdx = -1
			list.UnselectAll()
			list.Refresh()
			onSave(routers)
		}, w)
	})
	deleteBtn.Importance = widget.DangerImportance

	toolbar := container.NewGridWithColumns(3, addBtn, editBtn, deleteBtn)
	content := container.NewBorder(toolbar, nil, nil, nil, list)
	w.SetContent(content)
	w.Show()
}

// showRouterForm shows an add/edit router dialog.
func showRouterForm(
	a fyne.App,
	parent fyne.Window,
	existing *RouterConfig,
	allRouters []RouterConfig,
	onConfirm func(RouterConfig),
) {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Home Router")
	addressEntry := widget.NewEntry()
	addressEntry.SetPlaceHolder("192.168.1.1")
	loginEntry := widget.NewEntry()
	loginEntry.SetPlaceHolder("admin")
	passwordEntry := widget.NewPasswordEntry()

	originalName := ""
	if existing != nil {
		originalName = existing.Name
		nameEntry.SetText(existing.Name)
		addressEntry.SetText(existing.Address)
		loginEntry.SetText(existing.Login)
		if pw := getPassword(existing); pw != "" {
			passwordEntry.SetText(pw)
		}
	}

	errorLabel := widget.NewLabel("")
	errorLabel.Wrapping = fyne.TextWrapWord

	form := widget.NewForm(
		widget.NewFormItem("Name", nameEntry),
		widget.NewFormItem("Address", addressEntry),
		widget.NewFormItem("Login", loginEntry),
		widget.NewFormItem("Password", passwordEntry),
	)

	content := container.NewVBox(form, errorLabel)

	var dlg dialog.Dialog
	dlg = dialog.NewCustomConfirm("Router", "Save", "Cancel", content, func(save bool) {
		if !save {
			return
		}

		name := strings.TrimSpace(nameEntry.Text)
		address := strings.TrimSuffix(strings.TrimSpace(addressEntry.Text), "/")
		login := strings.TrimSpace(loginEntry.Text)
		password := passwordEntry.Text

		if name == "" || address == "" || login == "" || password == "" {
			errorLabel.SetText("Please fill in all fields.")
			showRouterFormWithValues(a, parent, existing, allRouters, onConfirm,
				nameEntry.Text, addressEntry.Text, loginEntry.Text, passwordEntry.Text,
				"Please fill in all fields.")
			return
		}
		for _, r := range allRouters {
			if r.Name == name && name != originalName {
				showRouterFormWithValues(a, parent, existing, allRouters, onConfirm,
					nameEntry.Text, addressEntry.Text, loginEntry.Text, passwordEntry.Text,
					"A router with this name already exists.")
				return
			}
		}

		dlg.Hide()
		checkDlg := dialog.NewCustomWithoutButtons("Connecting...",
			container.NewVBox(
				widget.NewLabel("Verifying router connection..."),
				widget.NewProgressBarInfinite(),
			), parent)
		checkDlg.Show()

		go func() {
			router := NewKeeneticRouter(address, login, password, name)
			err := router.Login()
			checkDlg.Hide()

			if err != nil {
				showRouterFormWithValues(a, parent, existing, allRouters, onConfirm,
					name, address, login, password,
					"Connection failed: "+err.Error())
				return
			}

			cfg := RouterConfig{Name: name, Address: address, Login: login}
			if ip, e := router.GetNetworkIP(); e == nil && ip != "" {
				cfg.NetworkIP = ip
			}
			if urls, e := router.GetKeenDNSURLs(); e == nil {
				cfg.KeenDNS = urls
			}
			setPassword(&cfg, password)
			onConfirm(cfg)
		}()
	}, parent)

	dlg.Resize(fyne.NewSize(380, 340))
	dlg.Show()
}

// showRouterFormWithValues re-opens the form pre-filled with an error.
func showRouterFormWithValues(
	a fyne.App,
	parent fyne.Window,
	existing *RouterConfig,
	allRouters []RouterConfig,
	onConfirm func(RouterConfig),
	name, address, login, password, errMsg string,
) {
	stub := &RouterConfig{Name: name, Address: address, Login: login, Password: password}
	if existing != nil {
		stub.NetworkIP = existing.NetworkIP
		stub.KeenDNS = existing.KeenDNS
	}
	originalName := ""
	if existing != nil {
		originalName = existing.Name
	}

	nameEntry := widget.NewEntry()
	nameEntry.SetText(name)
	addressEntry := widget.NewEntry()
	addressEntry.SetText(address)
	loginEntry := widget.NewEntry()
	loginEntry.SetText(login)
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetText(password)

	errorLabel := widget.NewLabel(errMsg)
	errorLabel.Wrapping = fyne.TextWrapWord

	form := widget.NewForm(
		widget.NewFormItem("Name", nameEntry),
		widget.NewFormItem("Address", addressEntry),
		widget.NewFormItem("Login", loginEntry),
		widget.NewFormItem("Password", passwordEntry),
	)

	var dlg dialog.Dialog
	dlg = dialog.NewCustomConfirm("Router", "Save", "Cancel",
		container.NewVBox(form, errorLabel),
		func(save bool) {
			if !save {
				return
			}
			n := strings.TrimSpace(nameEntry.Text)
			addr := strings.TrimSuffix(strings.TrimSpace(addressEntry.Text), "/")
			lg := strings.TrimSpace(loginEntry.Text)
			pw := passwordEntry.Text

			if n == "" || addr == "" || lg == "" || pw == "" {
				showRouterFormWithValues(a, parent, existing, allRouters, onConfirm,
					n, addr, lg, pw, "Please fill in all fields.")
				return
			}
			for _, r := range allRouters {
				if r.Name == n && n != originalName {
					showRouterFormWithValues(a, parent, existing, allRouters, onConfirm,
						n, addr, lg, pw, "A router with this name already exists.")
					return
				}
			}

			dlg.Hide()
			checkDlg := dialog.NewCustomWithoutButtons("Connecting...",
				container.NewVBox(
					widget.NewLabel("Verifying router connection..."),
					widget.NewProgressBarInfinite(),
				), parent)
			checkDlg.Show()

			go func() {
				router := NewKeeneticRouter(addr, lg, pw, n)
				err := router.Login()
				checkDlg.Hide()

				if err != nil {
					showRouterFormWithValues(a, parent, existing, allRouters, onConfirm,
						n, addr, lg, pw, "Connection failed: "+err.Error())
					return
				}

				cfg := RouterConfig{Name: n, Address: addr, Login: lg}
				if stub.NetworkIP != "" {
					cfg.NetworkIP = stub.NetworkIP
				} else if ip, e := router.GetNetworkIP(); e == nil && ip != "" {
					cfg.NetworkIP = ip
				}
				if ip, e := router.GetNetworkIP(); e == nil && ip != "" {
					cfg.NetworkIP = ip
				}
				if urls, e := router.GetKeenDNSURLs(); e == nil {
					cfg.KeenDNS = urls
				}
				setPassword(&cfg, pw)
				onConfirm(cfg)
			}()
		}, parent)

	dlg.Resize(fyne.NewSize(380, 360))
	dlg.Show()
}
