package app

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"frf-tui/internal/client"
)

const (
	minListWidth        = 36
	listPercent         = 0.42
	panelPaddingX       = 1
	panelPaddingY       = 0
	headerHeight        = 7
	panelBorder         = 1
	detailHeaderHeight  = 1
	detailPollInterval  = 10 * time.Second
	detailUpdateFlash   = 3 * time.Second
	commentPreviewCount = 5
	commentPageStep     = 20
)

var bannerLines = []string{
	" _____              _____             _ ",
	"|  ___| __ ___  ___|  ___|__  ___  __| |",
	"| |_ | '__/ _ \\/ _ \\ |_ / _ \\/ _ \\/ _` |",
	"|  _|| | |  __/  __/  _|  __/  __/ (_| |",
	"|_|  |_|  \\___|\\___|_|  \\___|\\___|\\__,_|",
}

var tagRegex = regexp.MustCompile(`#\S+`)
var linkRegex = regexp.MustCompile(`(?i)\b(?:https?://|www\.)[^\s]+`)

type Model struct {
	width       int
	height      int
	status      string
	err         error
	config      Config
	posts       []client.Post
	offset      int
	loadingMore bool
	noMorePosts bool

	list    list.Model
	detail  viewport.Model
	listW   int
	detailW int
	ready   bool
	focused pane

	activeSection        string
	showGroups           bool
	lastDetailPostID     string
	lastDetailLikesDelta int
	detailUpdatedAt      time.Time
	detailLoading        bool
	spinnerFrame         int
	commentsViewActive   bool
	commentsFetchLimit   int
}

type pane int

const (
	paneList pane = iota
	paneDetail
)

type timelineMsg struct {
	posts  []client.Post
	append bool
}

type detailPollMsg struct{}

type detailPostMsg struct {
	post  client.Post
	found bool
}

type errMsg struct {
	err error
}

type postItem struct {
	post client.Post
}

func (p postItem) FilterValue() string {
	return strings.TrimSpace(p.post.Body)
}

func New() Model {
	config, err := LoadConfig()
	status := "loading timeline..."
	if err != nil {
		status = "configuration error"
	}

	activeSection := "Home"

	listModel := list.New([]list.Item{}, newPostDelegate(), 0, 0)
	listModel.Title = activeSection
	listModel.SetShowStatusBar(false)
	listModel.SetFilteringEnabled(false)
	listModel.SetShowHelp(true)
	listModel.Styles.Title = lipgloss.NewStyle().Bold(true)
	listModel.KeyMap.Quit.Unbind()

	detail := viewport.New(0, 0)
	detail.SetContent("Select a post to view details.")

	return Model{
		status:        status,
		err:           err,
		config:        config,
		list:          listModel,
		detail:        detail,
		focused:       paneList,
		activeSection: activeSection,
	}
}

func (m Model) Init() tea.Cmd {
	if m.err != nil {
		return nil
	}
	return tea.Batch(fetchTimelineCmd(m.config, 0, false), scheduleDetailPoll())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
	case timelineMsg:
		m.loadingMore = false
		if msg.append {
			if len(msg.posts) == 0 {
				m.noMorePosts = true
				return m, nil
			}
			m.noMorePosts = false
			m.posts = append(m.posts, msg.posts...)
			current := m.list.Index()
			m.list.SetItems(appendPostItems(m.list.Items(), msg.posts))
			m.list.Select(current)
			return m, nil
		}
		m.posts = msg.posts
		m.noMorePosts = len(m.posts) == 0
		if len(m.posts) == 0 {
			m.status = "no posts available"
		} else {
			m.status = ""
			m.list.SetItems(buildPostItems(m.posts))
			m.list.Select(0)
			m.refreshDetail()
		}
	case errMsg:
		m.err = msg.err
		m.status = ""
		m.loadingMore = false
	case detailPollMsg:
		cmd := scheduleDetailPoll()
		if m.focused != paneDetail {
			return m, cmd
		}
		item, ok := m.list.SelectedItem().(postItem)
		if !ok || item.post.ID == "" {
			return m, cmd
		}
		m.detailLoading = true
		m.spinnerFrame = (m.spinnerFrame + 1) % len(detailSpinnerFrames)
		return m, tea.Batch(cmd, fetchPostCmd(m.config, item.post.ID, m.currentMaxComments()))
	case detailPostMsg:
		m.detailLoading = false
		if !msg.found {
			return m, nil
		}
		if current, ok := findPostByID(m.posts, msg.post.ID); ok {
			if postContentChanged(current, msg.post) {
				m.detailUpdatedAt = time.Now()
			}
			m.lastDetailLikesDelta = likesDelta(current, msg.post)
		}
		updatePostInSlice(m.posts, msg.post)
		items, updated := updatePostItems(m.list.Items(), msg.post)
		if updated {
			m.list.SetItems(items)
		}
		if item, ok := m.list.SelectedItem().(postItem); ok && item.post.ID == msg.post.ID {
			m.refreshDetail()
		}
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyTab, tea.KeyShiftTab:
			m.toggleFocus()
			return m, nil
		case tea.KeyEsc:
			if m.commentsViewActive {
				m.commentsViewActive = false
				m.commentsFetchLimit = 0
				m.refreshDetail()
				return m, nil
			}
			if m.showGroups {
				m.showGroups = false
				return m, nil
			}
		default:
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "r":
				m.showGroups = !m.showGroups
				return m, nil
			case "a":
				m.focused = paneDetail
				m.commentsViewActive = true
				if m.commentsFetchLimit == 0 {
					m.commentsFetchLimit = commentPageStep
				}
				return m.fetchDetailIfNeeded()
			case "h":
				m.focused = paneList
				m.setActiveSection("Home")
			case "l":
				m.focused = paneDetail
			case "m":
				m.setActiveSection("Direct messages")
			case "D":
				m.setActiveSection("Discussions")
			case "s":
				m.setActiveSection("Saved posts")
			case "b":
				m.setActiveSection("Best of the day")
			case "n":
				m.setActiveSection("Notifications")
			}
		}
	}

	if m.showGroups {
		return m, nil
	}

	var cmd tea.Cmd
	if m.focused == paneList {
		m.list, cmd = m.list.Update(msg)
		m.refreshDetail()
		if m.shouldLoadMore() {
			m.loadingMore = true
			m.offset = len(m.posts)
			return m, tea.Batch(cmd, fetchTimelineCmd(m.config, m.offset, true))
		}
		return m, cmd
	}

	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "j", "down":
			m.detail.LineDown(1)
			return m, nil
		case "k", "up":
			if m.commentsViewActive && m.detail.AtTop() {
				if cmd := m.maybeLoadMoreComments(); cmd != nil {
					return m, cmd
				}
			}
			m.detail.LineUp(1)
			return m, nil
		case "pgdown":
			m.detail.LineDown(m.detail.Height)
			return m, nil
		case "pgup":
			if m.commentsViewActive && m.detail.AtTop() {
				if cmd := m.maybeLoadMoreComments(); cmd != nil {
					return m, cmd
				}
			}
			m.detail.LineUp(m.detail.Height)
			return m, nil
		}
	}

	m.detail, cmd = m.detail.Update(msg)
	return m, cmd
}

func selectedPostID(listModel list.Model) (string, bool) {
	item, ok := listModel.SelectedItem().(postItem)
	if !ok || item.post.ID == "" {
		return "", false
	}
	return item.post.ID, true
}

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}

	if m.status != "" {
		return m.status
	}

	if !m.ready {
		return "loading..."
	}

	header := m.renderHeader()

	bodyHeight := m.height - headerHeight
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	listUp := m.list.Index() > 0
	listDown := m.list.Index() < len(m.list.Items())-1
	detailUp := !m.detail.AtTop()
	detailDown := !m.detail.AtBottom()
	listTitle := formatPaneTitle(m.activeSection, m.focused == paneList, listUp, listDown)
	if m.list.Title != listTitle {
		m.list.Title = listTitle
	}

	leftStyle := paneStyle(m.focused == paneList).Width(m.listW).Height(bodyHeight).Padding(panelPaddingY, panelPaddingX)
	rightStyle := paneStyle(m.focused == paneDetail).Width(m.detailW).Height(bodyHeight).Padding(panelPaddingY, panelPaddingX)

	headerLabel := "Detail"
	if m.commentsViewActive {
		headerLabel = "Comments"
	}
	detailHeader := renderPaneHeader(headerLabel, m.focused == paneDetail, detailUp, detailDown, m.detail.Width)
	rightContent := lipgloss.JoinVertical(lipgloss.Top, detailHeader, m.detail.View())

	left := leftStyle.Render(m.list.View())
	right := rightStyle.Render(rightContent)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	content := lipgloss.JoinVertical(lipgloss.Top, header, body)

	if m.showGroups {
		return m.renderGroupsOverlay()
	}

	return content
}

func (m *Model) resize() {
	if m.width <= 0 || m.height <= 0 {
		return
	}

	listWidth := int(float64(m.width) * listPercent)
	if listWidth < minListWidth {
		listWidth = minListWidth
	}
	if listWidth > m.width-24 {
		listWidth = m.width - 24
	}
	if listWidth < minListWidth {
		listWidth = minListWidth
	}

	detailWidth := m.width - listWidth - 1
	if detailWidth < 24 {
		detailWidth = 24
		listWidth = m.width - detailWidth - 1
	}

	height := m.height - headerHeight
	if height < 6 {
		height = 6
	}

	m.listW = listWidth
	m.detailW = detailWidth
	listInnerWidth := listWidth - (panelPaddingX * 2) - (panelBorder * 2)
	listInnerHeight := height - (panelPaddingY * 2) - (panelBorder * 2)
	if listInnerWidth < 1 {
		listInnerWidth = 1
	}
	if listInnerHeight < 1 {
		listInnerHeight = 1
	}
	m.list.SetSize(listInnerWidth, listInnerHeight)

	detailInnerWidth := detailWidth - (panelPaddingX * 2) - (panelBorder * 2)
	detailInnerHeight := height - (panelPaddingY * 2) - (panelBorder * 2) - detailHeaderHeight
	if detailInnerWidth < 1 {
		detailInnerWidth = 1
	}
	if detailInnerHeight < 1 {
		detailInnerHeight = 1
	}
	m.detail.Width = detailInnerWidth
	m.detail.Height = detailInnerHeight
	m.ready = true
}

func paneStyle(focused bool) lipgloss.Style {
	borderColor := lipgloss.Color("240")
	if focused {
		borderColor = lipgloss.Color("75")
	}
	return lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(borderColor)
}

func formatPaneTitle(label string, focused, up, down bool) string {
	marker := " "
	if focused {
		marker = ">"
	}
	indicators := make([]string, 0, 2)
	if up {
		indicators = append(indicators, "^")
	}
	if down {
		indicators = append(indicators, "v")
	}
	if len(indicators) == 0 {
		return fmt.Sprintf("%s %s", marker, label)
	}
	return fmt.Sprintf("%s %s %s", marker, label, strings.Join(indicators, " "))
}

func renderPaneHeader(label string, focused, up, down bool, width int) string {
	if width < 1 {
		width = 1
	}
	text := formatPaneTitle(label, focused, up, down)
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	if focused {
		style = style.Foreground(lipgloss.Color("230")).Bold(true)
	}
	return style.Width(width).Render(text)
}

func (m Model) renderHeader() string {
	headerStyle := lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("230")).Padding(0, 1)
	bannerStyle := lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("229")).Bold(true)
	brand := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Render("FreeFeed")
	section := lipgloss.NewStyle().Foreground(lipgloss.Color("251")).Render(m.activeSection)
	search := lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Render("Search: [ / ]")
	divider := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("│")

	leftWidth := m.listW
	rightWidth := m.detailW
	if leftWidth <= 0 || rightWidth <= 0 {
		leftWidth = m.width
		rightWidth = 0
	}

	bannerRows := make([]string, 0, len(bannerLines))
	for i, line := range bannerLines {
		left := bannerStyle.Width(leftWidth).Align(lipgloss.Left).Render(line)
		right := ""
		if rightWidth > 0 && i == 0 {
			right = headerStyle.Width(rightWidth).Align(lipgloss.Right).Render(search)
		}
		if rightWidth > 0 {
			bannerRows = append(bannerRows, lipgloss.JoinHorizontal(lipgloss.Top, left, divider, right))
			continue
		}
		bannerRows = append(bannerRows, left)
	}
	left := lipgloss.JoinHorizontal(lipgloss.Top, brand, " ", section)
	row1Left := headerStyle.Width(leftWidth).Render(left)
	row1 := row1Left
	if rightWidth > 0 {
		row1Right := headerStyle.Width(rightWidth).Render("")
		row1 = lipgloss.JoinHorizontal(lipgloss.Top, row1Left, divider, row1Right)
	}

	hints := "h Home  m Direct  D Discussions  s Saved  b Best  n Notif  r Groups  q Quit"
	row2 := headerStyle.Width(m.width).Render(hints)

	rows := append(bannerRows, row1, row2)
	return lipgloss.JoinVertical(lipgloss.Top, rows...)
}

func (m *Model) setActiveSection(section string) {
	m.activeSection = section
	m.list.Title = section
}

func (m Model) renderGroupsOverlay() string {
	groups := []string{
		"Groups",
		"",
		"- homeautomat",
		"- FreeFeed Support",
		"- Life and kittens",
		"- MidiLibrary",
		"- Best of Hacker News",
		"",
		"Press r or Esc to close",
	}

	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
	content := box.Render(strings.Join(groups, "\n"))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m *Model) refreshDetail() {
	item, ok := m.list.SelectedItem().(postItem)
	if !ok {
		m.detail.SetContent("Select a post to view details.")
		return
	}
	if item.post.ID != m.lastDetailPostID {
		m.lastDetailPostID = item.post.ID
		m.lastDetailLikesDelta = 0
		m.detailUpdatedAt = time.Time{}
		m.detailLoading = false
		m.commentsViewActive = false
		m.commentsFetchLimit = 0
	}

	if m.commentsViewActive {
		m.refreshCommentsView(item.post)
		return
	}

	content := renderDetail(item.post, m.detail.Width, m.lastDetailLikesDelta, m.detailLoading, m.detailUpdatedAt, m.spinnerFrame)
	m.detail.SetContent(content)
}

func (m *Model) refreshCommentsView(post client.Post) {
	content := renderCommentsView(post, m.detail.Width, m.detailLoading)
	m.detail.SetContent(content)
}

func (m *Model) toggleFocus() {
	if m.focused == paneList {
		m.focused = paneDetail
	} else {
		m.focused = paneList
	}
}

func fetchTimelineCmd(config Config, offset int, append bool) tea.Cmd {
	return func() tea.Msg {
		client := client.New(config.BaseURL, config.Username, config.Password)
		if err := client.Authenticate(); err != nil {
			return errMsg{err: err}
		}

		posts, err := client.GetTimeline(config.TimelineType, config.Username, config.Limit, offset)
		if err != nil {
			return errMsg{err: err}
		}

		return timelineMsg{posts: posts, append: append}
	}
}

func fetchPostCmd(config Config, postID string, maxComments string) tea.Cmd {
	return func() tea.Msg {
		apiClient := client.New(config.BaseURL, config.Username, config.Password)
		if err := apiClient.Authenticate(); err != nil {
			return errMsg{err: err}
		}

		post, err := apiClient.GetPost(postID, maxComments)
		if err != nil {
			if errors.Is(err, client.ErrPostNotFound) {
				return detailPostMsg{found: false}
			}
			return errMsg{err: err}
		}

		return detailPostMsg{post: post, found: true}
	}
}

func scheduleDetailPoll() tea.Cmd {
	return tea.Tick(detailPollInterval, func(time.Time) tea.Msg {
		return detailPollMsg{}
	})
}

func appendPostItems(items []list.Item, posts []client.Post) []list.Item {
	if len(posts) == 0 {
		return items
	}

	appended := make([]list.Item, 0, len(items)+len(posts))
	appended = append(appended, items...)
	for _, post := range posts {
		appended = append(appended, postItem{post: post})
	}
	return appended
}

func updatePostItems(items []list.Item, post client.Post) ([]list.Item, bool) {
	updated := false
	for i, item := range items {
		postItemValue, ok := item.(postItem)
		if !ok {
			continue
		}
		if postItemValue.post.ID != post.ID {
			continue
		}
		items[i] = postItem{post: post}
		updated = true
	}
	return items, updated
}

func updatePostInSlice(posts []client.Post, post client.Post) {
	for i := range posts {
		if posts[i].ID == post.ID {
			posts[i] = post
			return
		}
	}
}

func findPostByID(posts []client.Post, id string) (client.Post, bool) {
	for _, post := range posts {
		if post.ID == id {
			return post, true
		}
	}
	return client.Post{}, false
}

func postContentChanged(oldPost, newPost client.Post) bool {
	if oldPost.Body != newPost.Body {
		return true
	}
	if commentsChanged(oldPost.Comments, newPost.Comments) {
		return true
	}
	return oldPost.LikesCount != newPost.LikesCount
}

func commentsChanged(oldComments, newComments []client.Comment) bool {
	if len(oldComments) != len(newComments) {
		return true
	}
	for i := range oldComments {
		if oldComments[i].ID != newComments[i].ID {
			return true
		}
		if oldComments[i].Body != newComments[i].Body {
			return true
		}
	}
	return false
}

func likesDelta(oldPost, newPost client.Post) int {
	if newPost.LikesCount <= oldPost.LikesCount {
		return 0
	}
	return newPost.LikesCount - oldPost.LikesCount
}

func (m Model) shouldLoadMore() bool {
	if m.loadingMore || m.noMorePosts {
		return false
	}
	if len(m.posts) == 0 {
		return false
	}
	return m.list.Index() >= len(m.posts)-1
}

func buildPostItems(posts []client.Post) []list.Item {
	items := make([]list.Item, 0, len(posts))
	for _, post := range posts {
		items = append(items, postItem{post: post})
	}
	return items
}

type postDelegate struct {
	selected lipgloss.Style
	normal   lipgloss.Style
}

func newPostDelegate() postDelegate {
	return postDelegate{
		selected: lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62")).PaddingLeft(1),
		normal:   lipgloss.NewStyle().Foreground(lipgloss.Color("252")).PaddingLeft(1),
	}
}

func (d postDelegate) Height() int {
	return 3
}

func (d postDelegate) Spacing() int {
	return 1
}

func (d postDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d postDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	post, ok := item.(postItem)
	if !ok {
		return
	}

	author := post.post.Author()
	width := m.Width() - 2
	if width < 20 {
		width = 20
	}
	body := trimBody(post.post.Body, width)
	meta := formatTimestamp(post.post.CreatedAt)

	line1 := fmt.Sprintf("%s", author)
	line2 := fmt.Sprintf("%s", body)
	line3 := fmt.Sprintf("%s", meta)

	style := d.normal
	if index == m.Index() {
		style = d.selected
	}

	content := strings.Join([]string{line1, line2, line3}, "\n")
	contentWidth := m.Width()
	if contentWidth < 1 {
		contentWidth = 1
	}
	fmt.Fprint(w, style.Width(contentWidth).Render(content))
}

func renderDetail(post client.Post, width int, likesDeltaCount int, loading bool, updatedAt time.Time, spinnerFrame int) string {
	if width <= 0 {
		width = 40
	}

	header := lipgloss.NewStyle().Bold(true)
	meta := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	body := lipgloss.NewStyle()

	bodyText := highlightTags(highlightLinks(wrapText(strings.TrimSpace(post.Body), width)))
	statusLine := detailActivityLine(loading, updatedAt, spinnerFrame)
	likesLine := formatLikesLine(post, 3, likesDeltaCount)
	lines := []string{
		header.Render(post.Author()),
		meta.Render(formatTimestamp(post.CreatedAt)),
		"",
	}
	if statusLine != "" {
		lines = append(lines, meta.Render(statusLine), "")
	}
	lines = append(lines,
		body.Render(bodyText),
	)
	if likesLine != "" {
		lines = append(lines, "", meta.Render(wrapText(likesLine, width)))
	}

	commentsBlock := renderComments(post, width, loading)
	if commentsBlock != "" {
		lines = append(lines, "", commentsBlock)
	}

	return strings.Join(lines, "\n")
}

func renderComments(post client.Post, width int, loading bool) string {
	if width <= 0 {
		width = 40
	}

	count := post.CommentsCount
	if count == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Comments: 0")
	}
	if !post.CommentsLoaded {
		note := "open detail to load"
		if loading {
			note = "loading..."
		}
		label := fmt.Sprintf("Comments (%d) - %s", count, note)
		return lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(label)
	}

	header := lipgloss.NewStyle().Bold(true)
	meta := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	body := lipgloss.NewStyle()

	previewCount := commentPreviewCount
	comments := post.Comments
	if len(comments) > previewCount {
		comments = comments[len(comments)-previewCount:]
	}
	commentHeader := fmt.Sprintf("Comments (%d)", count)
	lines := []string{header.Render(commentHeader)}
	if len(post.Comments) > previewCount {
		lines = append(lines, meta.Render("a - see all"))
	}
	for _, comment := range comments {
		author := comment.Author()
		timestamp := formatTimestamp(comment.CreatedAt)
		lineHeader := author
		if timestamp != "" {
			lineHeader = fmt.Sprintf("%s · %s", author, timestamp)
		}
		lines = append(lines, meta.Render(lineHeader))
		bodyText := highlightTags(highlightLinks(wrapText(strings.TrimSpace(comment.Body), width-2)))
		lines = append(lines, body.Render(indentLines(bodyText, "  ")))
		lines = append(lines, "")
	}
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return strings.Join(lines, "\n")
}

func renderCommentsView(post client.Post, width int, loading bool) string {
	if width <= 0 {
		width = 40
	}
	if post.CommentsCount == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Comments: 0")
	}
	if !post.CommentsLoaded {
		note := "loading..."
		label := fmt.Sprintf("Comments (%d) - %s", post.CommentsCount, note)
		return lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(label)
	}

	header := lipgloss.NewStyle().Bold(true)
	meta := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	body := lipgloss.NewStyle()

	lines := []string{header.Render("Comments · Esc to return")}
	lines = append(lines, meta.Render(fmt.Sprintf("Total: %d", post.CommentsCount)))
	if loading {
		lines = append(lines, meta.Render("updating..."))
	}
	if post.CommentsCount > len(post.Comments) {
		lines = append(lines, meta.Render("^ more above"))
	}
	for _, comment := range post.Comments {
		author := comment.Author()
		timestamp := formatTimestamp(comment.CreatedAt)
		lineHeader := author
		if timestamp != "" {
			lineHeader = fmt.Sprintf("%s · %s", author, timestamp)
		}
		lines = append(lines, meta.Render(lineHeader))
		bodyText := highlightTags(highlightLinks(wrapText(strings.TrimSpace(comment.Body), width-2)))
		lines = append(lines, body.Render(indentLines(bodyText, "  ")))
		lines = append(lines, "")
	}
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return strings.Join(lines, "\n")
}

func (m Model) fetchDetailIfNeeded() (tea.Model, tea.Cmd) {
	if m.detailLoading {
		return m, nil
	}
	item, ok := m.list.SelectedItem().(postItem)
	if !ok || item.post.ID == "" {
		return m, nil
	}
	if current, ok := findPostByID(m.posts, item.post.ID); ok && current.CommentsLoaded && !m.commentsViewActive {
		m.refreshDetail()
		return m, nil
	}
	m.detailLoading = true
	m.spinnerFrame = (m.spinnerFrame + 1) % len(detailSpinnerFrames)
	m.refreshDetail()
	return m, fetchPostCmd(m.config, item.post.ID, m.currentMaxComments())
}

func (m *Model) currentMaxComments() string {
	if m.commentsViewActive {
		if m.commentsFetchLimit <= 0 {
			m.commentsFetchLimit = commentPageStep
		}
		return strconv.Itoa(m.commentsFetchLimit)
	}
	return strconv.Itoa(commentPreviewCount)
}

func (m *Model) maybeLoadMoreComments() tea.Cmd {
	if m.detailLoading {
		return nil
	}
	item, ok := m.list.SelectedItem().(postItem)
	if !ok || item.post.ID == "" {
		return nil
	}
	post, ok := findPostByID(m.posts, item.post.ID)
	if !ok {
		return nil
	}
	if post.CommentsCount <= len(post.Comments) {
		return nil
	}
	if m.commentsFetchLimit <= 0 {
		m.commentsFetchLimit = commentPageStep
	}
	if post.CommentsCount > 0 && m.commentsFetchLimit >= post.CommentsCount {
		return nil
	}
	m.commentsFetchLimit += commentPageStep
	if post.CommentsCount > 0 && m.commentsFetchLimit > post.CommentsCount {
		m.commentsFetchLimit = post.CommentsCount
	}
	m.detailLoading = true
	m.spinnerFrame = (m.spinnerFrame + 1) % len(detailSpinnerFrames)
	m.refreshDetail()
	return fetchPostCmd(m.config, item.post.ID, m.currentMaxComments())
}

func indentLines(text, prefix string) string {
	if text == "" {
		return text
	}
	rows := strings.Split(text, "\n")
	for i := range rows {
		rows[i] = prefix + rows[i]
	}
	return strings.Join(rows, "\n")
}

func formatLikesLine(post client.Post, maxNames int, deltaCount int) string {
	if post.LikesCount <= 0 {
		return ""
	}
	if maxNames <= 0 {
		maxNames = 1
	}

	likes := post.Likes
	if len(likes) > maxNames {
		likes = likes[len(likes)-maxNames:]
	}

	names := make([]string, 0, len(likes))
	for _, user := range likes {
		name := formatUserDisplay(user)
		if name != "" {
			names = append(names, name)
		}
	}

	if len(names) == 0 {
		return fmt.Sprintf("%d people liked this", post.LikesCount)
	}

	remaining := post.LikesCount - len(names)
	list := strings.Join(names, ", ")
	delta := ""
	if deltaCount > 0 {
		if deltaCount == 1 {
			delta = " (+1 like)"
		} else {
			delta = fmt.Sprintf(" (+%d likes)", deltaCount)
		}
	}
	if remaining > 0 {
		return fmt.Sprintf("<3 %s and %d other people liked this%s", list, remaining, delta)
	}
	return fmt.Sprintf("<3 %s liked this%s", list, delta)
}

func formatUserDisplay(user client.User) string {
	if user.ScreenName != "" && user.Username != "" && user.ScreenName != user.Username {
		return fmt.Sprintf("%s (%s)", user.ScreenName, user.Username)
	}
	if user.ScreenName != "" {
		return user.ScreenName
	}
	return user.Username
}

func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	words := strings.Fields(strings.ReplaceAll(text, "\n", " "))
	if len(words) == 0 {
		return ""
	}

	var lines []string
	current := words[0]
	for _, word := range words[1:] {
		if len([]rune(current))+1+len([]rune(word)) <= width {
			current = current + " " + word
			continue
		}
		lines = append(lines, current)
		current = word
	}
	lines = append(lines, current)
	return strings.Join(lines, "\n")
}

var detailSpinnerFrames = []string{"-", "\\", "|", "/"}

func detailActivityLine(loading bool, updatedAt time.Time, spinnerFrame int) string {
	if loading {
		return fmt.Sprintf("%s updating", detailSpinnerFrames[spinnerFrame%len(detailSpinnerFrames)])
	}
	if updatedAt.IsZero() {
		return ""
	}
	if time.Since(updatedAt) > detailUpdateFlash {
		return ""
	}
	return ". updated"
}

func highlightTags(text string) string {
	if text == "" {
		return text
	}
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("75")).Bold(true)
	indices := tagRegex.FindAllStringIndex(text, -1)
	if len(indices) == 0 {
		return text
	}
	var builder strings.Builder
	last := 0
	for _, match := range indices {
		start := match[0]
		end := match[1]
		if start > last {
			builder.WriteString(text[last:start])
		}
		if isTagInURL(text, start, end) {
			builder.WriteString(text[start:end])
		} else {
			builder.WriteString(style.Render(text[start:end]))
		}
		last = end
	}
	if last < len(text) {
		builder.WriteString(text[last:])
	}
	return builder.String()
}

func highlightLinks(text string) string {
	if text == "" {
		return text
	}
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Background(lipgloss.Color("236")).Bold(true)
	return linkRegex.ReplaceAllStringFunc(text, func(raw string) string {
		link := raw
		trailing := ""
		for len(link) > 0 {
			last := link[len(link)-1]
			if strings.ContainsRune(".,:;!?)]}", rune(last)) {
				trailing = string(last) + trailing
				link = link[:len(link)-1]
				continue
			}
			break
		}
		if link == "" {
			return raw
		}
		return style.Render(link) + trailing
	})
}

func isTagInURL(text string, start, end int) bool {
	if start < 0 || end > len(text) || start >= end {
		return false
	}
	boundaryChars := " \n\t\r"
	chunkStart := 0
	if idx := strings.LastIndexAny(text[:start], boundaryChars); idx != -1 {
		chunkStart = idx + 1
	}
	chunkEnd := len(text)
	if idx := strings.IndexAny(text[end:], boundaryChars); idx != -1 {
		chunkEnd = end + idx
	}
	chunk := text[chunkStart:chunkEnd]
	return strings.Contains(chunk, "http://") || strings.Contains(chunk, "https://") || strings.Contains(chunk, "www.")
}

func trimBody(body string, limit int) string {
	trimmed := strings.TrimSpace(strings.ReplaceAll(body, "\n", " "))
	if trimmed == "" {
		return "(empty)"
	}
	if limit <= 0 {
		return trimmed
	}

	runes := []rune(trimmed)
	if len(runes) <= limit {
		return trimmed
	}

	if limit <= 3 {
		return string(runes[:limit])
	}
	return string(runes[:limit-3]) + "..."
}

func formatTimestamp(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	value, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return trimmed
	}

	var timestamp time.Time
	if value > 1_000_000_000_000 {
		timestamp = time.UnixMilli(value)
	} else {
		timestamp = time.Unix(value, 0)
	}

	return relativeTime(timestamp)
}

func relativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	ago := time.Since(t)
	if ago < 0 {
		ago = -ago
	}

	seconds := int(ago.Seconds())
	if seconds < 60 {
		return fmt.Sprintf("%ds ago", seconds)
	}
	minutes := int(ago.Minutes())
	if minutes < 60 {
		return fmt.Sprintf("%dm ago", minutes)
	}
	hours := int(ago.Hours())
	if hours < 24 {
		return fmt.Sprintf("%dh ago", hours)
	}
	days := hours / 24
	if days < 30 {
		return fmt.Sprintf("%dd ago", days)
	}
	months := days / 30
	if months < 12 {
		return fmt.Sprintf("%dmo ago", months)
	}
	years := months / 12
	return fmt.Sprintf("%dy ago", years)
}
