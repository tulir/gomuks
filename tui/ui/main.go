package ui

import "go.mau.fi/mauview"

type MainView struct {
	*mauview.Grid
	app *App
}

func NewMainView(app *App) *MainView {
	mv := &MainView{
		Grid: mauview.NewGrid(),
		app:  app,
	}

	mv.SetRows([]int{1})
	mv.SetColumns([]int{20, 120})

	mv.AddComponent(app.Views.RoomList, 0, 0, 20, 1)
	mv.AddComponent(app.Views.CurrentTimelineView, 20, 0, 100, 1)

	return mv
}
