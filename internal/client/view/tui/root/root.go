package root

import (
	"context"
	"gophkeeper/internal/client/auth"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/components/nav"
	"gophkeeper/internal/client/view/tui/views/auth/login"
	"gophkeeper/internal/client/view/tui/views/auth/register"
	"gophkeeper/internal/client/view/tui/views/auth/unlock"
	"gophkeeper/internal/client/view/tui/views/auth/welcome"
	"gophkeeper/internal/client/view/tui/views/cards/cardadd"
	"gophkeeper/internal/client/view/tui/views/cards/cardlist"
	fileslist "gophkeeper/internal/client/view/tui/views/files/filelist"
	filesupload "gophkeeper/internal/client/view/tui/views/files/fileupload"
	"gophkeeper/internal/client/view/tui/views/home"
	noteadd "gophkeeper/internal/client/view/tui/views/notes/noteadd"
	notelist "gophkeeper/internal/client/view/tui/views/notes/notelist"
	passwordadd "gophkeeper/internal/client/view/tui/views/passwords/passwordadd"
	passwordlist "gophkeeper/internal/client/view/tui/views/passwords/passwordlist"
	cardv1 "gophkeeper/internal/shared/proto/card/v1"
	filev1 "gophkeeper/internal/shared/proto/file/v1"
	notev1 "gophkeeper/internal/shared/proto/note/v1"
	passwordv1 "gophkeeper/internal/shared/proto/password/v1"
	userv1 "gophkeeper/internal/shared/proto/user/v1"

	tea "charm.land/bubbletea/v2"
)

// SessionStore loads and persists the current session; satisfied by the
// session repository.
type SessionStore interface {
	Get(ctx context.Context) (*auth.Session, error)
	Save(ctx context.Context, cred auth.Credentials) (*auth.Session, error)
	Clear(ctx context.Context) error
}

type rootModel struct {
	current tea.Model
	stack   []tea.Model
	Deps
}

type Deps struct {
	UserClient     userv1.UserServiceClient
	CardClient     cardv1.CardServiceClient
	PasswordClient passwordv1.PasswordServiceClient
	NoteClient     notev1.NoteServiceClient
	FileClient     filev1.FileServiceClient
	Vault          *vault.Vault
	SessionStore
}

func New(deps Deps) tea.Model {
	return rootModel{
		current: build(deps, resolveStart(deps)),
		Deps:    deps,
	}
}

func build(deps Deps, id nav.ScreenID) tea.Model {
	switch id {
	case nav.Login:
		return login.New(login.Prop{Client: deps.UserClient, Store: deps.SessionStore, Vault: deps.Vault})
	case nav.Register:
		return register.New(register.Prop{Client: deps.UserClient, Store: deps.SessionStore, Vault: deps.Vault})
	case nav.Unlock:
		session, err := deps.SessionStore.Get(context.Background())
		if err != nil {
			return welcome.New()
		}
		return unlock.New(deps.Vault, session)
	case nav.Home:
		return home.New()
	case nav.Cards:
		return cardlist.New(cardlist.Prop{Vault: deps.Vault, Client: deps.CardClient})
	case nav.CardAdd:
		return cardadd.New(cardadd.Prop{Vault: deps.Vault, Client: deps.CardClient})
	case nav.Passwords:
		return passwordlist.New(passwordlist.Prop{Vault: deps.Vault, Client: deps.PasswordClient})
	case nav.PasswordAdd:
		return passwordadd.New(passwordadd.Prop{Vault: deps.Vault, Client: deps.PasswordClient})
	case nav.Notes:
		return notelist.New(notelist.Prop{Vault: deps.Vault, Client: deps.NoteClient})
	case nav.NoteAdd:
		return noteadd.New(noteadd.Prop{Vault: deps.Vault, Client: deps.NoteClient})
	case nav.Files:
		return fileslist.New(fileslist.Prop{Vault: deps.Vault, Client: deps.FileClient})
	case nav.FileUpload:
		return filesupload.New(filesupload.Prop{Vault: deps.Vault, Client: deps.FileClient})
	default:
		return welcome.New()
	}
}

func resolveStart(deps Deps) nav.ScreenID {
	session, err := deps.SessionStore.Get(context.Background())
	if err != nil || session == nil {
		return nav.Welcome
	}
	if !session.Access.Expired(0) {
		if deps.Vault.Locked() {
			return nav.Unlock
		}
		return nav.Home
	}
	if session.Refresh.Raw == "" {
		return nav.Welcome
	}
	in := &userv1.RefreshRequest{}
	in.SetRefreshToken(session.Refresh.Raw)
	resp, err := deps.UserClient.Refresh(context.Background(), in)
	if err != nil {
		return nav.Welcome
	}
	if _, err := deps.SessionStore.Save(context.Background(), auth.Credentials{
		Login:        session.Login,
		AccessToken:  resp.GetAccessToken(),
		RefreshToken: resp.GetRefreshToken(),
		EncSalt:      session.EncSalt,
		WrappedDek:   session.WrappedDek,
	}); err != nil {
		return nav.Welcome
	}
	if deps.Vault.Locked() {
		return nav.Unlock
	}
	return nav.Home
}

func (m rootModel) Init() tea.Cmd {
	return m.current.Init()
}

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case nav.PushMsg:
		m.stack = append(m.stack, m.current)
		m.current = build(m.Deps, msg.ID)
		return m, m.current.Init()
	case nav.PushModelMsg:
		m.stack = append(m.stack, m.current)
		m.current = msg.Model
		return m, m.current.Init()
	case nav.ResetMsg:
		m.stack = nil
		m.current = build(m.Deps, msg.ID)
		return m, m.current.Init()
	case nav.BackMsg:
		if n := len(m.stack); n > 0 {
			m.current = m.stack[n-1]
			m.stack = m.stack[:n-1]
		}
		return m, nil
	case nav.LogoutMsg:
		if err := m.SessionStore.Clear(context.Background()); err != nil {
			return m, nil
		}
		m.Vault.Lock()
		m.stack = nil
		m.current = build(m.Deps, nav.Welcome)
		return m, m.current.Init()
	}

	var cmd tea.Cmd
	m.current, cmd = m.current.Update(msg)
	return m, cmd
}

func (m rootModel) View() tea.View {
	return m.current.View()
}
