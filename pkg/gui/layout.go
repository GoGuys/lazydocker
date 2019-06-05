package gui

import (
	"github.com/jesseduffield/gocui"
	"github.com/jesseduffield/lazydocker/pkg/utils"
)

// getFocusLayout returns a manager function for when view gain and lose focus
func (gui *Gui) getFocusLayout() func(g *gocui.Gui) error {
	var previousView *gocui.View
	return func(g *gocui.Gui) error {
		newView := gui.g.CurrentView()
		if err := gui.onFocusChange(); err != nil {
			return err
		}
		// for now we don't consider losing focus to a popup panel as actually losing focus
		if newView != previousView && !gui.isPopupPanel(newView.Name()) {
			if err := gui.onFocusLost(previousView, newView); err != nil {
				return err
			}
			if err := gui.onFocus(newView); err != nil {
				return err
			}
			previousView = newView
		}
		return nil
	}
}

func (gui *Gui) onFocusChange() error {
	currentView := gui.g.CurrentView()
	for _, view := range gui.g.Views() {
		view.Highlight = view == currentView
	}
	return nil
}

func (gui *Gui) onFocusLost(v *gocui.View, newView *gocui.View) error {
	if v == nil {
		return nil
	}
	gui.Log.Info(v.Name() + " focus lost")
	return nil
}

func (gui *Gui) onFocus(v *gocui.View) error {
	if v == nil {
		return nil
	}
	gui.Log.Info(v.Name() + " focus gained")
	return nil
}

// layout is called for every screen re-render e.g. when the screen is resized
func (gui *Gui) layout(g *gocui.Gui) error {
	g.Highlight = true
	width, height := g.Size()

	information := gui.Config.Version

	minimumHeight := 9
	minimumWidth := 10
	if height < minimumHeight || width < minimumWidth {
		v, err := g.SetView("limit", 0, 0, width-1, height-1, 0)
		if err != nil {
			if err.Error() != "unknown view" {
				return err
			}
			v.Title = gui.Tr.SLocalize("NotEnoughSpace")
			v.Wrap = true
			_, _ = g.SetViewOnTop("limit")
		}
		return nil
	}

	currView := gui.g.CurrentView()
	currentCyclebleView := gui.State.PreviousView
	if currView != nil {
		viewName := currView.Name()
		usePreviouseView := true
		for _, view := range cyclableViews {
			if view == viewName {
				currentCyclebleView = viewName
				usePreviouseView = false
				break
			}
		}
		if usePreviouseView {
			currentCyclebleView = gui.State.PreviousView
		}
	}

	usableSpace := height - 4

	tallPanels := 3

	vHeights := map[string]int{
		"status":     tallPanels,
		"services":   usableSpace/tallPanels + usableSpace%tallPanels,
		"containers": usableSpace / tallPanels,
		"images":     usableSpace / tallPanels,
		"options":    1,
	}

	if height < 28 {
		defaultHeight := 3
		if height < 21 {
			defaultHeight = 1
		}
		vHeights = map[string]int{
			"status":     defaultHeight,
			"services":   defaultHeight,
			"containers": defaultHeight,
			"images":     defaultHeight,
			"options":    defaultHeight,
		}
		vHeights[currentCyclebleView] = height - defaultHeight*tallPanels - 1
	}

	optionsVersionBoundary := width - max(len(utils.Decolorise(information)), 1)
	leftSideWidth := width / 3

	appStatus := gui.statusManager.getStatusString()
	appStatusOptionsBoundary := 0
	if appStatus != "" {
		appStatusOptionsBoundary = len(appStatus) + 2
	}

	_, _ = g.SetViewOnBottom("limit")
	g.DeleteView("limit")

	v, err := g.SetView("main", leftSideWidth+1, 0, width-1, height-2, gocui.LEFT)
	if err != nil {
		if err.Error() != "unknown view" {
			return err
		}
		v.Wrap = true
		v.FgColor = gocui.ColorWhite
	}

	if v, err := g.SetView("status", 0, 0, leftSideWidth, vHeights["status"]-1, gocui.BOTTOM|gocui.RIGHT); err != nil {
		if err.Error() != "unknown view" {
			return err
		}
		v.Title = gui.Tr.SLocalize("StatusTitle")
		v.FgColor = gocui.ColorWhite
	}

	servicesView, err := g.SetViewBeneath("services", "status", vHeights["services"])
	if err != nil {
		if err.Error() != "unknown view" {
			return err
		}
		servicesView.Highlight = true
		servicesView.Title = gui.Tr.SLocalize("ServicesTitle")
		servicesView.FgColor = gocui.ColorWhite

		gui.focusPoint(0, gui.State.Panels.Services.SelectedLine, len(gui.DockerCommand.Services), servicesView)
	}

	containersView, err := g.SetViewBeneath("containers", "services", vHeights["containers"])
	if err != nil {
		if err.Error() != "unknown view" {
			return err
		}
		containersView.Highlight = true
		containersView.Title = gui.Tr.SLocalize("ContainersTitle")
		containersView.FgColor = gocui.ColorWhite

		gui.focusPoint(0, gui.State.Panels.Containers.SelectedLine, len(gui.DockerCommand.Containers), containersView)
	}

	imagesView, err := g.SetViewBeneath("images", "containers", vHeights["images"])
	if err != nil {
		if err.Error() != "unknown view" {
			return err
		}
		imagesView.Highlight = true
		imagesView.Title = gui.Tr.SLocalize("ImagesTitle")
		imagesView.FgColor = gocui.ColorWhite

		gui.focusPoint(0, gui.State.Panels.Images.SelectedLine, len(gui.DockerCommand.Images), imagesView)
	}

	if v, err := g.SetView("options", appStatusOptionsBoundary-1, height-2, optionsVersionBoundary-1, height, 0); err != nil {
		if err.Error() != "unknown view" {
			return err
		}
		v.Frame = false
		if v.FgColor, err = gui.GetOptionsPanelTextColor(); err != nil {
			return err
		}
	}

	if appStatusView, err := g.SetView("appStatus", -1, height-2, width, height, 0); err != nil {
		if err.Error() != "unknown view" {
			return err
		}
		appStatusView.BgColor = gocui.ColorDefault
		appStatusView.FgColor = gocui.ColorCyan
		appStatusView.Frame = false
		if _, err := g.SetViewOnBottom("appStatus"); err != nil {
			return err
		}
	}

	if v, err := g.SetView("information", optionsVersionBoundary-1, height-2, width, height, 0); err != nil {
		if err.Error() != "unknown view" {
			return err
		}
		v.BgColor = gocui.ColorDefault
		v.FgColor = gocui.ColorGreen
		v.Frame = false
		if err := gui.renderString(g, "information", information); err != nil {
			return err
		}

		// doing this here because it'll only happen once
		if err := gui.loadNewDirectory(); err != nil {
			return err
		}
	}

	if gui.g.CurrentView() == nil {
		if _, err := gui.g.SetCurrentView(gui.getContainersView().Name()); err != nil {
			return err
		}

		if err := gui.switchFocus(gui.g, nil, gui.getServicesView()); err != nil {
			return err
		}
	}

	type listViewState struct {
		selectedLine int
		lineCount    int
	}

	listViews := map[*gocui.View]listViewState{
		containersView: {selectedLine: gui.State.Panels.Containers.SelectedLine, lineCount: len(gui.DockerCommand.Containers)},
		imagesView:     {selectedLine: gui.State.Panels.Images.SelectedLine, lineCount: len(gui.DockerCommand.Images)},
	}

	// menu view might not exist so we check to be safe
	if menuView, err := gui.g.View("menu"); err == nil {
		listViews[menuView] = listViewState{selectedLine: gui.State.Panels.Menu.SelectedLine, lineCount: gui.State.MenuItemCount}
	}
	for view, state := range listViews {
		// check if the selected line is now out of view and if so refocus it
		if err := gui.focusPoint(0, state.selectedLine, state.lineCount, view); err != nil {
			return err
		}
	}

	// here is a good place log some stuff
	// if you download humanlog and do tail -f development.log | humanlog
	// this will let you see these branches as prettified json
	// gui.Log.Info(utils.AsJson(gui.State.Branches[0:4]))
	return gui.resizeCurrentPopupPanel(g)
}
