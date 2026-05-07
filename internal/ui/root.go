package ui

import (
	"fmt"
	"sort"
	"sync"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"technews-tui/internal/api"
	"technews-tui/internal/bookmark"
	"technews-tui/internal/browser"
	"technews-tui/internal/config"
	"technews-tui/internal/model"
)

type viewState int

const (
	stateList viewState = iota
	stateComments
	stateSettings
	stateBookmarks
)

// --- Messages ---

type postsLoadedMsg struct{ posts []model.Post }
type commentsLoadedMsg struct{ comments []model.Comment }
type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

// --- Root Model ---

type RootModel struct {
	state         viewState
	listModel     ListModel
	commentModel  CommentModel
	settingsModel SettingsModel
	bookmarkModel BookmarkModel
	bookmarkStore *bookmark.Store
	sources       []api.Source
	cfg           *config.Config
	allPosts      []model.Post // full unfiltered set
	sourceFilter  string       // "" = all, "hn", "reddit", etc.
	sourceNames   []string     // unique source IDs for cycling
	width         int
	height        int
	err           error
	loading       bool
	showHelp      bool
}

func NewRootModel(cfg *config.Config) RootModel {
	store := bookmark.NewStore(bookmark.DefaultPath())
	_ = store.Load()

	m := RootModel{
		state:         stateList,
		listModel:     NewListModel(store),
		bookmarkStore: store,
		cfg:           cfg,
		loading:       true,
	}
	m.rebuildSources()
	return m
}

func (m *RootModel) rebuildSources() {
	m.sources = []api.Source{}
	for id, sc := range m.cfg.Sources {
		if !sc.Enabled {
			continue
		}
		switch id {
		case "hn":
			m.sources = append(m.sources, api.NewHNClient())
		case "reddit":
			m.sources = append(m.sources, api.NewRedditClient(sc.Targets))
		case "lobsters":
			m.sources = append(m.sources, api.NewLobstersClient())
		case "lemmy":
			m.sources = append(m.sources, api.NewLemmyClient(sc.Targets))
		case "devto":
			m.sources = append(m.sources, api.NewDevToClient())
		case "rss":
			m.sources = append(m.sources, api.NewRSSClient(sc.Targets))
		}
	}
}

func (m RootModel) Init() tea.Cmd {
	return m.fetchPostsCmd()
}

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.listModel.SetSize(msg.Width, msg.Height)
		m.commentModel.SetSize(msg.Width, msg.Height)
		return m, nil

	case postsLoadedMsg:
		m.loading = false
		m.allPosts = msg.posts
		m.sourceNames = uniqueSources(msg.posts)
		m.applyFilter()
		return m, nil

	case commentsLoadedMsg:
		m.commentModel.SetComments(msg.comments)
		return m, nil

	case settingsDoneMsg:
		m.cfg = msg.config
		_ = config.Save(m.cfg)
		m.state = stateList
		m.loading = true
		m.sourceFilter = "" // reset filter when sources may have changed
		m.rebuildSources()
		return m, m.fetchPostsCmd()

	case bookmarkDoneMsg:
		m.state = stateList
		m.listModel.SetPosts(m.allPosts) // Triggers re-render of ★ indicator
		m.applyFilter()
		return m, nil

	case errMsg:
		m.err = msg.err
		m.loading = false
		return m, nil

	case tea.KeyMsg:
		// ctrl+c always quits
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		// ? toggles help overlay in any state
		if key.Matches(msg, keys.Help) {
			m.showHelp = !m.showHelp
			return m, nil
		}
		// esc also closes help if open
		if m.showHelp && key.Matches(msg, keys.Back) {
			m.showHelp = false
			return m, nil
		}
		// block all other keys when help is showing
		if m.showHelp {
			return m, nil
		}

		switch m.state {
		case stateList:
			return m.updateList(msg)
		case stateComments:
			return m.updateComments(msg)
		case stateSettings:
			return m.updateSettings(msg)
		case stateBookmarks:
			return m.updateBookmarks(msg)
		}
	}

	// Delegate non-key messages to the active view
	switch m.state {
	case stateList:
		var cmd tea.Cmd
		m.listModel, cmd = m.listModel.Update(msg)
		return m, cmd
	case stateComments:
		var cmd tea.Cmd
		m.commentModel, cmd = m.commentModel.Update(msg)
		return m, cmd
	case stateSettings:
		var cmd tea.Cmd
		m.settingsModel, cmd = m.settingsModel.Update(msg)
		return m, cmd
	case stateBookmarks:
		var cmd tea.Cmd
		var mdl tea.Model
		mdl, cmd = m.bookmarkModel.Update(msg)
		m.bookmarkModel = mdl.(BookmarkModel)
		return m, cmd
	}

	return m, nil
}

func (m RootModel) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit

	case msg.String() == "enter":
		post := m.listModel.SelectedPost()
		if post == nil {
			return m, nil
		}
		m.state = stateComments
		m.commentModel = NewCommentModel(*post)
		m.commentModel.SetBookmarked(m.bookmarkStore.Has(post.SourceURL))
		m.commentModel.SetSize(m.width, m.height)
		return m, m.fetchCommentsCmd(*post)

	case key.Matches(msg, keys.Open):
		post := m.listModel.SelectedPost()
		if post == nil {
			return m, nil
		}
		url := post.URL
		if url == "" {
			url = post.SourceURL
		}
		browser.Open(m.cfg.Browser, url) //nolint:errcheck
		return m, nil

	case key.Matches(msg, keys.Comments):
		post := m.listModel.SelectedPost()
		if post == nil {
			return m, nil
		}
		browser.Open(m.cfg.Browser, post.SourceURL) //nolint:errcheck
		return m, nil

	case key.Matches(msg, keys.Bookmark):
		post := m.listModel.SelectedPost()
		if post == nil {
			return m, nil
		}
		b := bookmark.Bookmark{
			Kind:         "post",
			Title:        post.Title,
			URL:          post.URL,
			SourceURL:    post.SourceURL,
			Source:       post.Source,
			SourceLabel:  post.SourceLabel,
			SourceID:     post.SourceID,
			Author:       post.Author,
			Points:       post.Points,
			CommentCount: post.CommentCount,
		}
		_, _ = m.bookmarkStore.Toggle(b)
		// Force list refresh to update ★
		m.listModel.SetPosts(m.listModel.posts)
		return m, nil

	case key.Matches(msg, keys.Bookmarks):
		m.state = stateBookmarks
		m.bookmarkModel = NewBookmarkModel(m.bookmarkStore, m.cfg)
		m.bookmarkModel.SetSize(m.width, m.height)
		return m, nil

	case key.Matches(msg, keys.Settings):
		m.state = stateSettings
		m.settingsModel = NewSettingsModel(m.cfg)
		m.settingsModel.SetSize(m.width, m.height)
		return m, nil

	case key.Matches(msg, keys.Filter):
		m.cycleFilter()
		return m, nil

	case key.Matches(msg, keys.Refresh):
		m.loading = true
		return m, m.fetchPostsCmd()
	}

	var cmd tea.Cmd
	m.listModel, cmd = m.listModel.Update(msg)
	return m, cmd
}

func (m RootModel) updateSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.settingsModel, cmd = m.settingsModel.Update(msg)
	return m, cmd
}

func (m RootModel) updateBookmarks(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var mdl tea.Model
	mdl, cmd = m.bookmarkModel.Update(msg)
	m.bookmarkModel = mdl.(BookmarkModel)
	return m, cmd
}

func (m RootModel) updateComments(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Let search mode intercept keys first
	if m.commentModel.searchMode != searchModeNone {
		var cmd tea.Cmd
		m.commentModel, cmd = m.commentModel.Update(msg)
		return m, cmd
	}

	switch {
	case key.Matches(msg, keys.Back), key.Matches(msg, keys.Quit):
		m.state = stateList
		m.listModel.SetPosts(m.listModel.posts) // Re-render in case bookmark changed
		return m, nil

	case key.Matches(msg, keys.Open):
		url := m.commentModel.post.URL
		if url == "" {
			url = m.commentModel.post.SourceURL
		}
		browser.Open(m.cfg.Browser, url) //nolint:errcheck
		return m, nil

	case key.Matches(msg, keys.Comments):
		browser.Open(m.cfg.Browser, m.commentModel.post.SourceURL) //nolint:errcheck
		return m, nil

	case key.Matches(msg, keys.Bookmark):
		b := bookmark.Bookmark{
			Kind:         "post",
			Title:        m.commentModel.post.Title,
			URL:          m.commentModel.post.URL,
			SourceURL:    m.commentModel.post.SourceURL,
			Source:       m.commentModel.post.Source,
			SourceLabel:  m.commentModel.post.SourceLabel,
			SourceID:     m.commentModel.post.SourceID,
			Author:       m.commentModel.post.Author,
			Points:       m.commentModel.post.Points,
			CommentCount: m.commentModel.post.CommentCount,
		}
		added, _ := m.bookmarkStore.Toggle(b)
		m.commentModel.SetBookmarked(added)
		return m, nil
	}

	var cmd tea.Cmd
	m.commentModel, cmd = m.commentModel.Update(msg)
	return m, cmd
}

func (m RootModel) View() string {
	if m.loading {
		return "\n  Loading stories..."
	}
	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n  Press r to retry.", m.err)
	}

	if m.showHelp {
		switch m.state {
		case stateComments:
			return renderHelp("Comment View", commentHelpEntries, m.width, m.height)
		case stateSettings:
			return renderHelp("Settings", settingsHelpEntries, m.width, m.height)
		case stateBookmarks:
			return renderHelp("Bookmarks", bookmarkHelpEntries, m.width, m.height)
		default:
			return renderHelp("Tech News TUI", listHelpEntries, m.width, m.height)
		}
	}

	switch m.state {
	case stateComments:
		return m.commentModel.View()
	case stateSettings:
		return m.settingsModel.View()
	case stateBookmarks:
		return m.bookmarkModel.View()
	default:
		return m.listModel.View()
	}
}

// --- Commands ---

func (m RootModel) fetchPostsCmd() tea.Cmd {
	return func() tea.Msg {
		var allPosts []model.Post
		var mu sync.Mutex
		var wg sync.WaitGroup

		limit := 30

		for _, s := range m.sources {
			wg.Add(1)
			go func(source api.Source) {
				defer wg.Done()
				sort := m.cfg.Sources[source.ID()].Sort

				posts, err := source.FetchPosts(sort, limit)
				if err != nil {
					return
				}
				mu.Lock()
				allPosts = append(allPosts, posts...)
				mu.Unlock()
			}(s)
		}

		wg.Wait()

		// Deduplicate by URL
		seenURLs := make(map[string]bool)
		var uniquePosts []model.Post
		for _, p := range allPosts {
			if p.URL != "" {
				if seenURLs[p.URL] {
					continue
				}
				seenURLs[p.URL] = true
			}
			uniquePosts = append(uniquePosts, p)
		}

		// Sort by rank, then preserve source order
		sort.SliceStable(uniquePosts, func(i, j int) bool {
			return uniquePosts[i].Rank < uniquePosts[j].Rank
		})

		return postsLoadedMsg{uniquePosts}
	}
}

// --- Source filter helpers ---

func (m *RootModel) applyFilter() {
	if m.sourceFilter == "" {
		m.listModel.SetPosts(m.allPosts)
		m.listModel.SetTitle("Tech News")
	} else {
		var filtered []model.Post
		for _, p := range m.allPosts {
			if p.Source == m.sourceFilter {
				filtered = append(filtered, p)
			}
		}
		m.listModel.SetPosts(filtered)
		m.listModel.SetTitle("Tech News [" + m.sourceFilter + "]")
	}

	var sorts []SortInfo
	for _, s := range m.sources {
		sorts = append(sorts, SortInfo{
			Name: s.Name(),
			Sort: m.cfg.Sources[s.ID()].Sort,
		})
	}
	m.listModel.SetSortInfo(sorts)
}

func (m *RootModel) cycleFilter() {
	if len(m.sourceNames) == 0 {
		return
	}
	if m.sourceFilter == "" {
		m.sourceFilter = m.sourceNames[0]
	} else {
		found := false
		for i, s := range m.sourceNames {
			if s == m.sourceFilter {
				if i+1 < len(m.sourceNames) {
					m.sourceFilter = m.sourceNames[i+1]
				} else {
					m.sourceFilter = "" // wrap back to All
				}
				found = true
				break
			}
		}
		if !found {
			m.sourceFilter = ""
		}
	}
	m.applyFilter()
}

func uniqueSources(posts []model.Post) []string {
	seen := map[string]bool{}
	var sources []string
	for _, p := range posts {
		if !seen[p.Source] {
			seen[p.Source] = true
			sources = append(sources, p.Source)
		}
	}
	sort.Strings(sources)
	return sources
}

func (m RootModel) fetchCommentsCmd(post model.Post) tea.Cmd {
	return func() tea.Msg {
		var comments []model.Comment
		var err error

		var source api.Source
		for _, s := range m.sources {
			if s.ID() == post.Source {
				source = s
				break
			}
		}

		if source == nil {
			return errMsg{fmt.Errorf("source not found: %s", post.Source)}
		}

		comments, err = source.FetchComments(post, 3)

		if err != nil {
			return errMsg{err}
		}
		return commentsLoadedMsg{comments}
	}
}
