package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	username   string
	password   string
	authToken  string
	httpClient *http.Client
}

var ErrPostNotFound = errors.New("post not found")

type User struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	ScreenName string `json:"screenName"`
}

type Post struct {
	ID             string
	Body           string
	CreatedAt      string
	User           User
	Likes          []User
	LikesCount     int
	Comments       []Comment
	CommentsCount  int
	CommentsLoaded bool
}

type Comment struct {
	ID         string
	Body       string
	CreatedAt  string
	User       User
	LikesCount int
	SeqNumber  int
}

type rawPost struct {
	ID                    string          `json:"id"`
	CreatedBy             string          `json:"createdBy"`
	Body                  string          `json:"body"`
	CreatedAt             string          `json:"createdAt"`
	Likes                 json.RawMessage `json:"likes"`
	LikesCount            int             `json:"likesCount"`
	Comments              []string        `json:"comments"`
	OmittedComments       int             `json:"omittedComments"`
	OmittedCommentsOffset int             `json:"omittedCommentsOffset"`
	CommentCount          int             `json:"commentCount"`
	CommentsCount         int             `json:"commentsCount"`
}

type rawComment struct {
	ID         string          `json:"id"`
	PostID     string          `json:"postId"`
	CreatedBy  string          `json:"createdBy"`
	Body       string          `json:"body"`
	CreatedAt  string          `json:"createdAt"`
	Likes      json.RawMessage `json:"likes"`
	LikesCount int             `json:"likesCount"`
	SeqNumber  int             `json:"seqNumber"`
}

type timeline struct {
	Posts []string `json:"posts"`
}

type timelineResponse struct {
	Posts     json.RawMessage `json:"posts"`
	Users     json.RawMessage `json:"users"`
	Timelines json.RawMessage `json:"timelines"`
}

type authResponse struct {
	AuthToken string                 `json:"authToken"`
	Users     map[string]interface{} `json:"users"`
}

func New(baseURL, username, password string) *Client {
	trimmed := strings.TrimRight(baseURL, "/")
	return &Client{
		baseURL:    trimmed,
		username:   username,
		password:   password,
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

func NewWithToken(baseURL, token string) *Client {
	trimmed := strings.TrimRight(baseURL, "/")
	return &Client{
		baseURL:    trimmed,
		authToken:  token,
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *Client) SetToken(token string) {
	c.authToken = token
}

func (c *Client) IsAuthenticated() bool {
	return c.authToken != ""
}

func (c *Client) Authenticate() error {
	payload := map[string]string{
		"username": c.username,
		"password": c.password,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var response authResponse
	if err := c.doJSON(ctx, http.MethodPost, "session", payload, &response, false); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	if response.AuthToken != "" {
		c.authToken = response.AuthToken
		return nil
	}

	if response.Users != nil {
		if token, ok := response.Users["authToken"].(string); ok && token != "" {
			c.authToken = token
			return nil
		}
	}

	return errors.New("authentication response missing auth token")
}

func (c *Client) GetTimeline(timelineType, username string, limit, offset int) ([]Post, error) {
	path, err := timelinePath(timelineType, username)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		params.Set("offset", fmt.Sprintf("%d", offset))
	}
	if encoded := params.Encode(); encoded != "" {
		path = path + "?" + encoded
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var response timelineResponse
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &response, true); err != nil {
		return nil, err
	}

	return flattenTimeline(response, timelineType), nil
}

func (c *Client) GetPost(postID string, maxComments string) (Post, error) {
	if postID == "" {
		return Post{}, errors.New("post id required")
	}

	path := fmt.Sprintf("posts/%s", url.PathEscape(postID))
	if maxComments != "" {
		path = path + "?maxComments=" + url.QueryEscape(maxComments)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var payload map[string]json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &payload, true); err != nil {
		if strings.Contains(err.Error(), "http error 404") {
			return Post{}, ErrPostNotFound
		}
		return Post{}, err
	}

	users := decodeUsers(payload["users"])
	postsByID := decodePosts(payload["posts"])
	commentsByID := decodeComments(payload["comments"])
	if len(postsByID) > 0 {
		if raw, ok := postsByID[postID]; ok {
			return buildPostWithComments(postID, raw, users, commentsByID), nil
		}
		for id, raw := range postsByID {
			return buildPostWithComments(id, raw, users, commentsByID), nil
		}
	}
	if raw, ok := decodeSinglePost(payload["posts"]); ok {
		id := resolvePostID(postID, raw)
		return buildPostWithComments(id, raw, users, commentsByID), nil
	}

	if raw, ok := decodeSinglePost(payload["post"]); ok {
		id := resolvePostID(postID, raw)
		return buildPostWithComments(id, raw, users, commentsByID), nil
	}

	return Post{}, ErrPostNotFound
}

func timelinePath(timelineType, username string) (string, error) {
	switch timelineType {
	case "home":
		return "timelines/home", nil
	case "discussions":
		return "timelines/filter/discussions", nil
	case "directs":
		return "timelines/filter/directs", nil
	case "posts", "likes", "comments":
		if username == "" {
			return "", errors.New("username required for user timeline")
		}
		escaped := url.PathEscape(username)
		if timelineType == "posts" {
			return fmt.Sprintf("timelines/%s", escaped), nil
		}
		return fmt.Sprintf("timelines/%s/%s", escaped, timelineType), nil
	default:
		return "", fmt.Errorf("unsupported timeline type: %s", timelineType)
	}
}

func flattenTimeline(response timelineResponse, timelineType string) []Post {
	postsByID := decodePosts(response.Posts)
	if len(postsByID) == 0 {
		return nil
	}

	users := decodeUsers(response.Users)

	orderedIDs := pickTimelineIDs(response, timelineType)
	posts := make([]Post, 0, len(orderedIDs))
	for _, id := range orderedIDs {
		raw, ok := postsByID[id]
		if !ok {
			continue
		}
		posts = append(posts, buildPostFromRaw(id, raw, users))
	}

	if len(posts) > 0 {
		return posts
	}

	for id, raw := range postsByID {
		posts = append(posts, buildPostFromRaw(id, raw, users))
	}
	return posts
}

func decodePosts(raw json.RawMessage) map[string]rawPost {
	posts := make(map[string]rawPost)
	if len(raw) == 0 {
		return posts
	}

	var postMap map[string]rawPost
	if err := json.Unmarshal(raw, &postMap); err == nil {
		return postMap
	}

	var postList []rawPost
	if err := json.Unmarshal(raw, &postList); err == nil {
		for _, post := range postList {
			if post.ID != "" {
				posts[post.ID] = post
			}
		}
		return posts
	}

	return posts
}

func decodeSinglePost(raw json.RawMessage) (rawPost, bool) {
	if len(raw) == 0 {
		return rawPost{}, false
	}

	var post rawPost
	if err := json.Unmarshal(raw, &post); err != nil {
		return rawPost{}, false
	}
	if post.ID == "" && post.Body == "" {
		return rawPost{}, false
	}
	return post, true
}

func resolvePostID(key string, post rawPost) string {
	if post.ID != "" {
		return post.ID
	}
	return key
}

func buildPostFromRaw(id string, raw rawPost, users map[string]User) Post {
	user := users[raw.CreatedBy]
	likeIDs := decodeLikeIDs(raw.Likes)
	likesCount := raw.LikesCount
	if likesCount < len(likeIDs) {
		likesCount = len(likeIDs)
	}
	commentsCount := len(raw.Comments) + raw.OmittedComments
	if commentsCount == 0 {
		if raw.CommentCount > 0 {
			commentsCount = raw.CommentCount
		} else if raw.CommentsCount > 0 {
			commentsCount = raw.CommentsCount
		}
	}
	commentsLoaded := false
	if commentsCount == 0 {
		commentsLoaded = true
	} else if raw.OmittedComments == 0 && len(raw.Comments) > 0 {
		commentsLoaded = true
	}
	return Post{
		ID:             resolvePostID(id, raw),
		Body:           raw.Body,
		CreatedAt:      raw.CreatedAt,
		User:           user,
		Likes:          resolveLikeUsers(users, likeIDs),
		LikesCount:     likesCount,
		CommentsCount:  commentsCount,
		CommentsLoaded: commentsLoaded,
	}
}

func buildPostWithComments(id string, raw rawPost, users map[string]User, comments map[string]rawComment) Post {
	post := buildPostFromRaw(id, raw, users)
	post.Comments = collectPostComments(post.ID, raw, users, comments)
	if len(post.Comments) > post.CommentsCount {
		post.CommentsCount = len(post.Comments)
	}
	if post.CommentsCount == 0 || len(post.Comments) == post.CommentsCount {
		post.CommentsLoaded = true
	}
	return post
}

func decodeUsers(raw json.RawMessage) map[string]User {
	users := make(map[string]User)
	if len(raw) == 0 {
		return users
	}

	var userMap map[string]User
	if err := json.Unmarshal(raw, &userMap); err == nil {
		return userMap
	}

	var userList []User
	if err := json.Unmarshal(raw, &userList); err == nil {
		for _, user := range userList {
			key := user.ID
			if key == "" {
				key = user.Username
			}
			if key != "" {
				users[key] = user
			}
		}
		return users
	}

	return users
}

func pickTimelineIDs(response timelineResponse, timelineType string) []string {
	if len(response.Timelines) == 0 {
		return nil
	}

	var timelineMap map[string]json.RawMessage
	if err := json.Unmarshal(response.Timelines, &timelineMap); err != nil {
		var timelineList []string
		if err := json.Unmarshal(response.Timelines, &timelineList); err == nil {
			return timelineList
		}
		return nil
	}

	if timelineType != "" {
		if raw, ok := timelineMap[timelineType]; ok {
			if ids := decodeTimelineIDs(raw); len(ids) > 0 {
				return ids
			}
		}
	}

	for _, raw := range timelineMap {
		if ids := decodeTimelineIDs(raw); len(ids) > 0 {
			return ids
		}
	}

	return nil
}

func decodeTimelineIDs(raw json.RawMessage) []string {
	var timelineData timeline
	if err := json.Unmarshal(raw, &timelineData); err == nil {
		return timelineData.Posts
	}

	var ids []string
	if err := json.Unmarshal(raw, &ids); err == nil {
		return ids
	}

	return nil
}

func decodeLikeIDs(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}

	var list []string
	if err := json.Unmarshal(raw, &list); err == nil {
		return list
	}

	var boolMap map[string]bool
	if err := json.Unmarshal(raw, &boolMap); err == nil {
		ids := make([]string, 0, len(boolMap))
		for id := range boolMap {
			ids = append(ids, id)
		}
		return ids
	}

	var anyMap map[string]interface{}
	if err := json.Unmarshal(raw, &anyMap); err == nil {
		ids := make([]string, 0, len(anyMap))
		for id := range anyMap {
			ids = append(ids, id)
		}
		return ids
	}

	return nil
}

func decodeComments(raw json.RawMessage) map[string]rawComment {
	comments := make(map[string]rawComment)
	if len(raw) == 0 {
		return comments
	}

	var commentMap map[string]rawComment
	if err := json.Unmarshal(raw, &commentMap); err == nil {
		return commentMap
	}

	var commentList []rawComment
	if err := json.Unmarshal(raw, &commentList); err == nil {
		for _, comment := range commentList {
			if comment.ID != "" {
				comments[comment.ID] = comment
			}
		}
		return comments
	}

	return comments
}

func collectPostComments(postID string, raw rawPost, users map[string]User, comments map[string]rawComment) []Comment {
	if len(comments) == 0 {
		return nil
	}

	if len(raw.Comments) > 0 {
		ordered := make([]Comment, 0, len(raw.Comments))
		for _, id := range raw.Comments {
			rawComment, ok := comments[id]
			if !ok {
				continue
			}
			ordered = append(ordered, buildCommentFromRaw(id, rawComment, users))
		}
		return ordered
	}

	filtered := make([]Comment, 0, len(comments))
	for id, rawComment := range comments {
		if rawComment.PostID != "" && rawComment.PostID != postID {
			continue
		}
		filtered = append(filtered, buildCommentFromRaw(id, rawComment, users))
	}
	if len(filtered) == 0 {
		return nil
	}
	if len(filtered) == 1 {
		return filtered
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].SeqNumber > 0 && filtered[j].SeqNumber > 0 {
			return filtered[i].SeqNumber < filtered[j].SeqNumber
		}
		left := parseTimestamp(filtered[i].CreatedAt)
		right := parseTimestamp(filtered[j].CreatedAt)
		if left != right {
			return left < right
		}
		return filtered[i].ID < filtered[j].ID
	})

	return filtered
}

func buildCommentFromRaw(id string, raw rawComment, users map[string]User) Comment {
	user := users[raw.CreatedBy]
	likesCount := raw.LikesCount
	if likesCount == 0 && len(raw.Likes) > 0 {
		likesCount = len(decodeLikeIDs(raw.Likes))
	}
	return Comment{
		ID:         resolveCommentID(id, raw),
		Body:       raw.Body,
		CreatedAt:  raw.CreatedAt,
		User:       user,
		LikesCount: likesCount,
		SeqNumber:  raw.SeqNumber,
	}
}

func resolveCommentID(key string, comment rawComment) string {
	if comment.ID != "" {
		return comment.ID
	}
	return key
}

func parseTimestamp(raw string) int64 {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0
	}
	value, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func resolveLikeUsers(users map[string]User, ids []string) []User {
	if len(ids) == 0 {
		return nil
	}

	likes := make([]User, 0, len(ids))
	for _, id := range ids {
		if user, ok := users[id]; ok {
			likes = append(likes, user)
			continue
		}
		likes = append(likes, User{ID: id, Username: id})
	}
	return likes
}

// SearchPosts searches for posts matching a query.
// Supports operators: from:, intitle:, incomment:, AND, OR.
func (c *Client) SearchPosts(query string, limit, offset int) ([]Post, error) {
	if query == "" {
		return nil, errors.New("query required")
	}

	params := url.Values{}
	params.Set("qs", query)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	path := "search?" + params.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var response timelineResponse
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &response, true); err != nil {
		return nil, err
	}

	return flattenTimeline(response, ""), nil
}

// WhoAmI returns the current authenticated user profile.
func (c *Client) WhoAmI() (UserProfile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var raw json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, "users/whoami", nil, &raw, true); err != nil {
		return UserProfile{}, err
	}

	return decodeUserProfileRaw(raw)
}

// GetUserProfile returns a user's profile by username.
func (c *Client) GetUserProfile(username string) (UserProfile, error) {
	if username == "" {
		return UserProfile{}, errors.New("username required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	path := fmt.Sprintf("users/%s", url.PathEscape(username))
	var raw json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &raw, true); err != nil {
		return UserProfile{}, err
	}

	return decodeUserProfileRaw(raw)
}

// GetSubscribers returns a user's subscribers (followers).
func (c *Client) GetSubscribers(username string) ([]User, error) {
	if username == "" {
		return nil, errors.New("username required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	path := fmt.Sprintf("users/%s/subscribers", url.PathEscape(username))
	var raw json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &raw, true); err != nil {
		return nil, err
	}

	return decodeUserListRaw(raw), nil
}

// GetSubscriptions returns a user's subscriptions (following).
func (c *Client) GetSubscriptions(username string) ([]User, error) {
	if username == "" {
		return nil, errors.New("username required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	path := fmt.Sprintf("users/%s/subscriptions", url.PathEscape(username))
	var raw json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &raw, true); err != nil {
		return nil, err
	}

	return decodeUserListRaw(raw), nil
}

// GetMyGroups returns groups the current user is a member of.
func (c *Client) GetMyGroups() ([]User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var raw json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, "users/whoami", nil, &raw, true); err != nil {
		return nil, err
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, errors.New("could not decode whoami response")
	}

	// "subscribers" in whoami contains all related users with full data including type
	subsRaw, ok := payload["subscribers"]
	if !ok {
		return nil, nil
	}

	var allUsers []struct {
		ID         string `json:"id"`
		Username   string `json:"username"`
		ScreenName string `json:"screenName"`
		Type       string `json:"type"`
	}
	if err := json.Unmarshal(subsRaw, &allUsers); err != nil {
		return nil, nil
	}

	groups := make([]User, 0)
	for _, u := range allUsers {
		if u.Type == "group" {
			groups = append(groups, User{
				ID:         u.ID,
				Username:   u.Username,
				ScreenName: u.ScreenName,
			})
		}
	}
	return groups, nil
}

// --- Write methods ---

// CreatePost creates a new post. feeds is a list of usernames (for groups or directs).
// If feeds is empty, posts to the authenticated user's feed.
func (c *Client) CreatePost(body string, feeds []string) (Post, error) {
	if body == "" {
		return Post{}, errors.New("post body required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if len(feeds) == 0 {
		feeds = []string{c.username}
		if c.username == "" {
			profile, err := c.WhoAmI()
			if err != nil {
				return Post{}, fmt.Errorf("determine username: %w", err)
			}
			feeds = []string{profile.Username}
		}
	}

	payload := map[string]any{
		"post": map[string]any{"body": body},
		"meta": map[string]any{"feeds": feeds},
	}

	var raw json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, "posts", payload, &raw, true); err != nil {
		return Post{}, err
	}

	return decodePostResponse(raw)
}

// CreateDirectPost creates a direct message to recipients.
func (c *Client) CreateDirectPost(body string, recipients []string) (Post, error) {
	if len(recipients) == 0 {
		return Post{}, errors.New("at least one recipient required")
	}
	return c.CreatePost(body, recipients)
}

// UpdatePost updates a post body.
func (c *Client) UpdatePost(postID, body string) error {
	if postID == "" {
		return errors.New("post id required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	payload := map[string]any{
		"post": map[string]any{"body": body},
	}

	path := fmt.Sprintf("posts/%s", url.PathEscape(postID))
	return c.doJSON(ctx, http.MethodPut, path, payload, nil, true)
}

// DeletePost deletes a post.
func (c *Client) DeletePost(postID string) error {
	if postID == "" {
		return errors.New("post id required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	path := fmt.Sprintf("posts/%s", url.PathEscape(postID))
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, true)
}

// LikePost likes a post.
func (c *Client) LikePost(postID string) error {
	return c.postAction(postID, "like")
}

// UnlikePost removes a like from a post.
func (c *Client) UnlikePost(postID string) error {
	return c.postAction(postID, "unlike")
}

// HidePost hides a post from the feed.
func (c *Client) HidePost(postID string) error {
	return c.postAction(postID, "hide")
}

// UnhidePost unhides a post.
func (c *Client) UnhidePost(postID string) error {
	return c.postAction(postID, "unhide")
}

func (c *Client) postAction(postID, action string) error {
	if postID == "" {
		return errors.New("post id required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	path := fmt.Sprintf("posts/%s/%s", url.PathEscape(postID), action)
	return c.doJSON(ctx, http.MethodPost, path, nil, nil, true)
}

// AddComment adds a comment to a post.
func (c *Client) AddComment(postID, body string) (Comment, error) {
	if postID == "" {
		return Comment{}, errors.New("post id required")
	}
	if body == "" {
		return Comment{}, errors.New("comment body required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	payload := map[string]any{
		"comment": map[string]any{
			"body":   body,
			"postId": postID,
		},
	}

	var raw json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, "comments", payload, &raw, true); err != nil {
		return Comment{}, err
	}

	return decodeCommentResponse(raw)
}

// UpdateComment updates a comment body.
func (c *Client) UpdateComment(commentID, body string) error {
	if commentID == "" {
		return errors.New("comment id required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	payload := map[string]any{
		"comment": map[string]any{"body": body},
	}

	path := fmt.Sprintf("comments/%s", url.PathEscape(commentID))
	return c.doJSON(ctx, http.MethodPut, path, payload, nil, true)
}

// DeleteComment deletes a comment.
func (c *Client) DeleteComment(commentID string) error {
	if commentID == "" {
		return errors.New("comment id required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	path := fmt.Sprintf("comments/%s", url.PathEscape(commentID))
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, true)
}

// Subscribe subscribes to a user.
func (c *Client) Subscribe(username string) error {
	if username == "" {
		return errors.New("username required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	path := fmt.Sprintf("users/%s/subscribe", url.PathEscape(username))
	return c.doJSON(ctx, http.MethodPost, path, nil, nil, true)
}

// Unsubscribe unsubscribes from a user.
func (c *Client) Unsubscribe(username string) error {
	if username == "" {
		return errors.New("username required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	path := fmt.Sprintf("users/%s/unsubscribe", url.PathEscape(username))
	return c.doJSON(ctx, http.MethodPost, path, nil, nil, true)
}

func decodePostResponse(raw json.RawMessage) (Post, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return Post{}, errors.New("could not decode post response")
	}

	users := decodeUsers(payload["users"])

	// Try as map or array first
	postsByID := decodePosts(payload["posts"])
	for id, rawPost := range postsByID {
		return buildPostFromRaw(id, rawPost, users), nil
	}

	// Try as single post object (create post returns this)
	if rp, ok := decodeSinglePost(payload["posts"]); ok {
		id := resolvePostID("", rp)
		return buildPostFromRaw(id, rp, users), nil
	}

	return Post{}, errors.New("no post in response")
}

func decodeCommentResponse(raw json.RawMessage) (Comment, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return Comment{}, errors.New("could not decode comment response")
	}

	users := decodeUsers(payload["users"])

	// Try as map or array
	comments := decodeComments(payload["comments"])
	for id, rc := range comments {
		return buildCommentFromRaw(id, rc, users), nil
	}

	// Try as single comment object
	if commentsRaw, ok := payload["comments"]; ok {
		var rc rawComment
		if err := json.Unmarshal(commentsRaw, &rc); err == nil && rc.ID != "" {
			return buildCommentFromRaw(rc.ID, rc, users), nil
		}
	}

	return Comment{}, errors.New("no comment in response")
}

// UserProfile represents a detailed user profile.
type UserProfile struct {
	ID              string            `json:"id"`
	Username        string            `json:"username"`
	ScreenName      string            `json:"screenName"`
	Type            string            `json:"type"`
	Description     string            `json:"description"`
	IsPrivate       string            `json:"isPrivate"`
	IsProtected     bool              `json:"isProtected"`
	IsGone          bool              `json:"isGone"`
	SubscriberCount int               `json:"subscriberCount"`
	SubscriptionCount int             `json:"subscriptionCount"`
	Subscriptions   []SubscriptionRef `json:"subscriptions,omitempty"`
}

// SubscriptionRef is a reference to a subscription (user or group).
type SubscriptionRef struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	ScreenName string `json:"screenName"`
	Type       string `json:"type"`
}

func decodeUserProfileRaw(raw json.RawMessage) (UserProfile, error) {
	// First try as a map with known keys
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return UserProfile{}, errors.New("could not decode user profile")
	}

	var profile UserProfile

	// Try "users" key — whoami returns {users: {...}, subscriptions: [...]}
	if usersRaw, ok := payload["users"]; ok {
		// Could be a single object
		if err := json.Unmarshal(usersRaw, &profile); err == nil && profile.Username != "" {
			decodeSubscriptionsInto(&profile, payload)
			return profile, nil
		}
		// Could be a map of id->user or an array
		users := decodeUsers(usersRaw)
		for _, u := range users {
			profile = UserProfile{ID: u.ID, Username: u.Username, ScreenName: u.ScreenName}
			decodeSubscriptionsInto(&profile, payload)
			return profile, nil
		}
	}

	// Try "user" key
	if userRaw, ok := payload["user"]; ok {
		if err := json.Unmarshal(userRaw, &profile); err == nil && profile.Username != "" {
			decodeSubscriptionsInto(&profile, payload)
			return profile, nil
		}
	}

	// Try direct decode (top-level is the user)
	if err := json.Unmarshal(raw, &profile); err == nil && profile.Username != "" {
		return profile, nil
	}

	return UserProfile{}, errors.New("could not decode user profile")
}

func decodeSubscriptionsInto(profile *UserProfile, payload map[string]json.RawMessage) {
	if subRaw, ok := payload["subscriptions"]; ok {
		var subs []SubscriptionRef
		if err := json.Unmarshal(subRaw, &subs); err == nil {
			profile.Subscriptions = subs
		}
	}
}

func decodeUserListRaw(raw json.RawMessage) []User {
	// Try as map with known keys
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err == nil {
		for _, key := range []string{"subscribers", "subscriptions", "users"} {
			data, ok := payload[key]
			if !ok {
				continue
			}
			var users []User
			if err := json.Unmarshal(data, &users); err == nil {
				return users
			}
			userMap := decodeUsers(data)
			result := make([]User, 0, len(userMap))
			for _, u := range userMap {
				result = append(result, u)
			}
			if len(result) > 0 {
				return result
			}
		}
	}

	// Try as direct array
	var users []User
	if err := json.Unmarshal(raw, &users); err == nil {
		return users
	}

	return nil
}

func (p Post) Author() string {
	if p.User.Username != "" {
		return p.User.Username
	}
	if p.User.ScreenName != "" {
		return p.User.ScreenName
	}
	return "unknown"
}

func (c Comment) Author() string {
	if c.User.Username != "" {
		return c.User.Username
	}
	if c.User.ScreenName != "" {
		return c.User.ScreenName
	}
	return "unknown"
}

func (c *Client) doJSON(ctx context.Context, method, path string, payload any, target any, auth bool) error {
	requestURL := c.baseURL + "/v4/" + strings.TrimLeft(path, "/")

	var bodyReader *bytes.Reader
	if payload != nil {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode payload: %w", err)
		}
		bodyReader = bytes.NewReader(payloadBytes)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth {
		if c.authToken == "" {
			return errors.New("missing auth token")
		}
		req.Header.Set("X-Authentication-Token", c.authToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("http error %d", resp.StatusCode)
	}

	if target == nil {
		return nil
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
