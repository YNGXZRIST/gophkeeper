package root

import (
	"context"
	"fmt"
	"gophkeeper/internal/client/auth"
	"gophkeeper/internal/client/repository"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/components/nav"
	"gophkeeper/internal/client/view/tui/components/theme"
	"gophkeeper/internal/client/view/tui/views/auth/login"
	"gophkeeper/internal/client/view/tui/views/auth/register"
	"gophkeeper/internal/client/view/tui/views/auth/unlock"
	"gophkeeper/internal/client/view/tui/views/auth/welcome"
	"gophkeeper/internal/client/view/tui/views/cards/cardadd"
	"gophkeeper/internal/client/view/tui/views/cards/cardconflict"
	"gophkeeper/internal/client/view/tui/views/cards/cardlist"
	"gophkeeper/internal/client/view/tui/views/files/fileconflict"
	fileslist "gophkeeper/internal/client/view/tui/views/files/filelist"
	filesupload "gophkeeper/internal/client/view/tui/views/files/fileupload"
	"gophkeeper/internal/client/view/tui/views/home"
	noteadd "gophkeeper/internal/client/view/tui/views/notes/noteadd"
	"gophkeeper/internal/client/view/tui/views/notes/noteconflict"
	notelist "gophkeeper/internal/client/view/tui/views/notes/notelist"
	passwordadd "gophkeeper/internal/client/view/tui/views/passwords/passwordadd"
	"gophkeeper/internal/client/view/tui/views/passwords/passwordconflict"
	passwordlist "gophkeeper/internal/client/view/tui/views/passwords/passwordlist"
	filev1 "gophkeeper/internal/shared/proto/file/v1"
	userv1 "gophkeeper/internal/shared/proto/user/v1"
	"log/slog"
	"strings"

	tea "charm.land/bubbletea/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SessionStore loads and persists the current session; satisfied by the
// session repository.
type SessionStore interface {
	Get(ctx context.Context) (*auth.Session, error)
	Save(ctx context.Context, cred auth.Credentials) (*auth.Session, error)
	Clear(ctx context.Context) error
}

type Syncer interface {
	SyncAll(ctx context.Context) error
}

type Logger interface {
	Error(msg string, args ...any)
}

type conflictCount struct {
	label string
	n     int
}

type syncDoneMsg struct {
	err       error
	offline   bool
	conflicts []conflictCount
}

type rootModel struct {
	current   tea.Model
	stack     []tea.Model
	syncErr   string
	offline   bool
	conflicts []conflictCount
	Deps
}

type Deps struct {
	UserClient    userv1.UserServiceClient
	Vault         *vault.Vault
	NotesRepo     *repository.EntryRepo
	PasswordsRepo *repository.EntryRepo
	CardsRepo     *repository.EntryRepo
	FilesRepo     *repository.FilesRepo
	FileClient    filev1.FileServiceClient
	Sync          Syncer
	Logger        Logger
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
		return cardlist.New(cardlist.Prop{Vault: deps.Vault, Repo: deps.CardsRepo})
	case nav.CardAdd:
		return cardadd.New(cardadd.Prop{Vault: deps.Vault, Repo: deps.CardsRepo})
	case nav.Passwords:
		return passwordlist.New(passwordlist.Prop{Vault: deps.Vault, Repo: deps.PasswordsRepo})
	case nav.PasswordAdd:
		return passwordadd.New(passwordadd.Prop{Vault: deps.Vault, Repo: deps.PasswordsRepo})
	case nav.Notes:
		return notelist.New(notelist.Prop{Vault: deps.Vault, Repo: deps.NotesRepo})
	case nav.NoteAdd:
		return noteadd.New(noteadd.Prop{Vault: deps.Vault, Repo: deps.NotesRepo})
	case nav.Sync:
		return noteconflict.New(noteconflict.Prop{Vault: deps.Vault, Repo: deps.NotesRepo, Sync: deps.Sync})
	case nav.CardSync:
		return cardconflict.New(cardconflict.Prop{Vault: deps.Vault, Repo: deps.CardsRepo, Sync: deps.Sync})
	case nav.PasswordSync:
		return passwordconflict.New(passwordconflict.Prop{Vault: deps.Vault, Repo: deps.PasswordsRepo, Sync: deps.Sync})
	case nav.FileSync:
		return fileconflict.New(fileconflict.Prop{Vault: deps.Vault, Repo: deps.FilesRepo, Sync: deps.Sync})
	case nav.Files:
		return fileslist.New(fileslist.Prop{Vault: deps.Vault, Repo: deps.FilesRepo, Client: deps.FileClient})
	case nav.FileUpload:
		return filesupload.New(filesupload.Prop{Vault: deps.Vault, Client: deps.FileClient, Repo: deps.FilesRepo})
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
	return tea.Batch(m.current.Init(), m.syncCmd())
}

func syncOnEnter(id nav.ScreenID) bool {
	switch id {
	case nav.Notes, nav.Cards, nav.Passwords, nav.Files:
		return true
	default:
		return false
	}
}

func isOffline(err error) bool {
	switch status.Code(err) {
	case codes.Unavailable, codes.DeadlineExceeded:
		return true
	default:
		return false
	}
}

func (m rootModel) syncCmd() tea.Cmd {
	sync := m.Sync
	if sync == nil {
		return nil
	}
	log := m.Logger
	listers := m.conflictListers()
	return func() tea.Msg {
		var done syncDoneMsg
		if err := sync.SyncAll(context.Background()); err != nil {
			if log != nil {
				log.Error("sync all", slog.Any("error", err))
			}
			if isOffline(err) {
				done.offline = true
			} else {
				done.err = err
			}
		}
		for _, l := range listers {
			rows, err := l.lister.ListConflicts(context.Background())
			if err != nil {
				if log != nil {
					log.Error("list conflicts", slog.Any("error", err))
				}
				continue
			}
			if len(rows) > 0 {
				done.conflicts = append(done.conflicts, conflictCount{label: l.label, n: len(rows)})
			}
		}
		return done
	}
}

type conflictLister interface {
	ListConflicts(ctx context.Context) ([]repository.ConflictRow, error)
}

type labeledLister struct {
	label  string
	lister conflictLister
}

func (m rootModel) conflictListers() []labeledLister {
	var out []labeledLister
	if m.NotesRepo != nil {
		out = append(out, labeledLister{label: home.LabelNotes, lister: m.NotesRepo})
	}
	if m.CardsRepo != nil {
		out = append(out, labeledLister{label: home.LabelCards, lister: m.CardsRepo})
	}
	if m.PasswordsRepo != nil {
		out = append(out, labeledLister{label: home.LabelPasswords, lister: m.PasswordsRepo})
	}
	if m.FilesRepo != nil {
		out = append(out, labeledLister{label: home.LabelFiles, lister: m.FilesRepo})
	}
	return out
}

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case nav.PushMsg:
		m.stack = append(m.stack, m.current)
		m.current = build(m.Deps, msg.ID)
		if syncOnEnter(msg.ID) {
			return m, tea.Batch(m.current.Init(), m.syncCmd())
		}
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
	case nav.SyncNowMsg:
		return m, m.syncCmd()
	case syncDoneMsg:
		if msg.err != nil {
			m.syncErr = msg.err.Error()
		} else {
			m.syncErr = ""
		}
		m.offline = msg.offline
		m.conflicts = msg.conflicts
		return m, nav.Reload()
	}

	var cmd tea.Cmd
	m.current, cmd = m.current.Update(msg)
	return m, cmd
}

func (m rootModel) View() tea.View {
	v := m.current.View()
	if m.Vault == nil || m.Vault.Locked() {
		return v
	}
	var footer []string
	if m.offline {
		footer = append(footer, theme.Blurred.Render("○ offline mode"))
	}
	if len(m.conflicts) > 0 {
		parts := make([]string, 0, len(m.conflicts))
		for _, c := range m.conflicts {
			parts = append(parts, fmt.Sprintf("%s: %d", c.label, c.n))
		}
		footer = append(footer, theme.Error.Render("⚠ conflicts — "+strings.Join(parts, ", ")+" — press c in the list"))
	}
	if m.syncErr != "" {
		footer = append(footer, theme.Error.Render("Sync error: "+m.syncErr))
	}
	if len(footer) > 0 {
		v.Content += "\n" + strings.Join(footer, "\n")
	}
	return v
}
