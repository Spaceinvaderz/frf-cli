package client

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("load fixture %s: %v", name, err)
	}
	return data
}

func testServer(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := NewWithToken(srv.URL, "test-token")
	return c
}

// --- Auth ---

func TestAuthenticate(t *testing.T) {
	fixture := loadFixture(t, "auth.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v4/session" {
			t.Errorf("expected /v4/session, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	}))
	defer srv.Close()

	c := New(srv.URL, "testuser", "testpass")
	err := c.Authenticate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !c.IsAuthenticated() {
		t.Error("expected authenticated")
	}
}

func TestAuthenticate_BadCredentials(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := New(srv.URL, "bad", "creds")
	err := c.Authenticate()
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Timeline ---

func TestGetTimeline(t *testing.T) {
	fixture := loadFixture(t, "timeline.json")
	c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("X-Authentication-Token") != "test-token" {
			t.Error("missing auth token")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	})

	posts, err := c.GetTimeline("home", "", 20, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(posts))
	}

	if posts[0].ID != "post-1" {
		t.Errorf("expected post-1, got %s", posts[0].ID)
	}
	if posts[0].User.Username != "alice" {
		t.Errorf("expected alice, got %s", posts[0].User.Username)
	}
	if posts[0].Body != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", posts[0].Body)
	}
	if posts[0].LikesCount != 1 {
		t.Errorf("expected 1 like, got %d", posts[0].LikesCount)
	}
	if posts[1].ID != "post-2" {
		t.Errorf("expected post-2, got %s", posts[1].ID)
	}
}

func TestGetTimeline_Empty(t *testing.T) {
	c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"posts":[],"users":[],"timelines":{}}`))
	})

	posts, err := c.GetTimeline("home", "", 20, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 0 {
		t.Errorf("expected 0 posts, got %d", len(posts))
	}
}

// --- Post ---

func TestGetPost(t *testing.T) {
	fixture := loadFixture(t, "post.json")
	c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v4/posts/post-1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	})

	post, err := c.GetPost("post-1", "all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if post.ID != "post-1" {
		t.Errorf("expected post-1, got %s", post.ID)
	}
	if post.Body != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", post.Body)
	}
	if len(post.Comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(post.Comments))
	}
	if post.Comments[0].Body != "Nice post!" {
		t.Errorf("expected 'Nice post!', got %q", post.Comments[0].Body)
	}
	if post.Comments[1].Body != "Thanks!" {
		t.Errorf("expected 'Thanks!', got %q", post.Comments[1].Body)
	}
}

func TestGetPost_NotFound(t *testing.T) {
	c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"err":"not found"}`))
	})

	_, err := c.GetPost("nonexistent", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if err != ErrPostNotFound {
		t.Errorf("expected ErrPostNotFound, got %v", err)
	}
}

// --- Search ---

func TestSearchPosts(t *testing.T) {
	fixture := loadFixture(t, "search.json")
	c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("qs") != "test query" {
			t.Errorf("expected qs=test query, got %s", r.URL.Query().Get("qs"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	})

	posts, err := c.SearchPosts("test query", 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	if posts[0].Body != "Searchable content here" {
		t.Errorf("unexpected body: %q", posts[0].Body)
	}
}

func TestSearchPosts_EmptyQuery(t *testing.T) {
	c := NewWithToken("http://localhost", "token")
	_, err := c.SearchPosts("", 10, 0)
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}

// --- User ---

func TestWhoAmI(t *testing.T) {
	fixture := loadFixture(t, "whoami.json")
	c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v4/users/whoami" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	})

	profile, err := c.WhoAmI()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.Username != "testuser" {
		t.Errorf("expected testuser, got %s", profile.Username)
	}
	if profile.ScreenName != "Test User" {
		t.Errorf("expected Test User, got %s", profile.ScreenName)
	}
}

func TestGetUserProfile(t *testing.T) {
	fixture := loadFixture(t, "user_profile.json")
	c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v4/users/bob" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	})

	profile, err := c.GetUserProfile("bob")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.Username != "bob" {
		t.Errorf("expected bob, got %s", profile.Username)
	}
	if profile.IsPrivate != "1" {
		t.Errorf("expected private, got %s", profile.IsPrivate)
	}
}

func TestGetSubscribers(t *testing.T) {
	fixture := loadFixture(t, "subscribers.json")
	c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	})

	users, err := c.GetSubscribers("alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if users[0].Username != "alice" {
		t.Errorf("expected alice, got %s", users[0].Username)
	}
}

// --- Groups ---

func TestGetMyGroups(t *testing.T) {
	fixture := loadFixture(t, "whoami.json")
	c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	})

	groups, err := c.GetMyGroups()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Username != "testgroup" {
		t.Errorf("expected testgroup, got %s", groups[0].Username)
	}
}

// --- Write operations ---

func TestCreatePost(t *testing.T) {
	fixture := loadFixture(t, "create_post.json")
	c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("expected Content-Type application/json")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	})
	c.username = "testuser"

	post, err := c.CreatePost("New post body", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if post.ID != "new-post-1" {
		t.Errorf("expected new-post-1, got %s", post.ID)
	}
	if post.Body != "New post body" {
		t.Errorf("expected 'New post body', got %q", post.Body)
	}
}

func TestCreatePost_EmptyBody(t *testing.T) {
	c := NewWithToken("http://localhost", "token")
	_, err := c.CreatePost("", nil)
	if err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestAddComment(t *testing.T) {
	fixture := loadFixture(t, "create_comment.json")
	c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	})

	comment, err := c.AddComment("post-1", "New comment")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comment.ID != "new-comment-1" {
		t.Errorf("expected new-comment-1, got %s", comment.ID)
	}
	if comment.Body != "New comment" {
		t.Errorf("expected 'New comment', got %q", comment.Body)
	}
}

func TestDeletePost(t *testing.T) {
	c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/v4/posts/post-1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})

	err := c.DeletePost("post-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLikePost(t *testing.T) {
	c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v4/posts/post-1/like" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})

	err := c.LikePost("post-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSubscribe(t *testing.T) {
	c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v4/users/alice/subscribe" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})

	err := c.Subscribe("alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Error cases ---

func TestServerError(t *testing.T) {
	c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := c.GetTimeline("home", "", 20, 0)
	if err == nil {
		t.Fatal("expected error on 500")
	}
}

func TestNoAuthToken(t *testing.T) {
	c := New("http://localhost/v4", "", "")
	_, err := c.GetTimeline("home", "", 20, 0)
	if err == nil {
		t.Fatal("expected error without auth")
	}
}
