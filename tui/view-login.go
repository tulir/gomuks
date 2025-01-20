package tui

import (
	"github.com/gdamore/tcell/v2"
	"go.mau.fi/mauview"
	"maunium.net/go/gomuks/config"
)

type LoginView struct {
	*mauview.Form

	container *mauview.Centerer

	homeserverLabel *mauview.TextField
	idLabel   *mauview.TextField
	passwordLabel   *mauview.TextField

	homeserver *mauview.InputField
	id   *mauview.InputField
	password   *mauview.InputField
	error      *mauview.TextView

	loginButton *mauview.Button
	quitButton  *mauview.Button

	loading bool

	config *config.Config
	parent *GomuksTUI
}

func (gt *GomuksTUI) NewLoginView() mauview.Component {
	view := &LoginView{
		Form: mauview.NewForm(),

		idLabel:   mauview.NewTextField().SetText("User ID"),
		passwordLabel:   mauview.NewTextField().SetText("Password"),
		homeserverLabel: mauview.NewTextField().SetText("Homeserver"),

		id:   mauview.NewInputField(),
		password:   mauview.NewInputField(),
		homeserver: mauview.NewInputField(),

		loginButton: mauview.NewButton("Login"),
		quitButton:  mauview.NewButton("Quit"),

		parent: gt,
	}

	view.homeserver.SetPlaceholder("https://example.com").SetText("").SetTextColor(tcell.ColorWhite)
	view.id.SetPlaceholder("@user:example.com").SetText("").SetTextColor(tcell.ColorWhite)
	view.password.SetPlaceholder("correct horse battery staple").SetMaskCharacter('*').SetTextColor(tcell.ColorWhite)

	view.quitButton.
		SetOnClick(gt.App.ForceStop).
		SetBackgroundColor(tcell.ColorDarkCyan).
		SetForegroundColor(tcell.ColorWhite).
		SetFocusedForegroundColor(tcell.ColorWhite)
	view.loginButton.
		SetBackgroundColor(tcell.ColorDarkCyan).
		SetForegroundColor(tcell.ColorWhite).
		SetFocusedForegroundColor(tcell.ColorWhite)

	view.
		SetColumns([]int{1, 10, 1, 30, 1}).
		SetRows([]int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1})
	view.
		AddFormItem(view.id, 3, 1, 1, 1).
		AddFormItem(view.password, 3, 3, 1, 1).
		AddFormItem(view.homeserver, 3, 5, 1, 1).
		AddFormItem(view.loginButton, 1, 7, 3, 1).
		AddFormItem(view.quitButton, 1, 9, 3, 1).
		AddComponent(view.idLabel, 1, 1, 1, 1).
		AddComponent(view.passwordLabel, 1, 3, 1, 1).
		AddComponent(view.homeserverLabel, 1, 5, 1, 1)
	view.FocusNextItem()
	gt.loginView = view

	view.container = mauview.Center(mauview.NewBox(view).SetTitle("Log in to Matrix"), 45, 13)
	view.container.SetAlwaysFocusChild(true)
	return view.container

}
