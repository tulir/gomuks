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

	mv.SetRows([]int{50, 50})
	mv.SetColumns([]int{20, 120})

	mv.AddComponent(app.Views.RoomList, 0, 0, 20, 1)
	mv.AddComponent(app.Views.CurrentTimelineView, 1, 0, 50, 1)

	return mv
}
