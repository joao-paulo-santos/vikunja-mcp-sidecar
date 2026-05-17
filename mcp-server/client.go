package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL:    baseURL,
		Token:      token,
		HTTPClient: &http.Client{},
	}
}

func (c *Client) doRequest(method, path string, body any) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

const (
	maxRetries     = 5
	retryBaseDelay = 50 * time.Millisecond
)

func isRetryable(statusCode int, body string) bool {
	if statusCode != 500 {
		return false
	}
	return strings.Contains(body, "database is locked")
}

func (c *Client) doRequestWithRetry(method, path string, body any) ([]byte, error) {
	for attempt := 0; attempt <= maxRetries; attempt++ {
		var reqBody io.Reader
		if body != nil {
			data, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("marshal body: %w", err)
			}
			reqBody = bytes.NewReader(data)
		}

		req, err := http.NewRequest(method, c.BaseURL+path, reqBody)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.Token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("execute request: %w", err)
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}

		if isRetryable(resp.StatusCode, string(respBody)) && attempt < maxRetries {
			delay := retryBaseDelay * time.Duration(1<<uint(attempt))
			time.Sleep(delay)
			continue
		}

		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
		}

		return respBody, nil
	}
	return nil, fmt.Errorf("API error: database is locked after %d retries", maxRetries)
}

func (c *Client) get(path string) ([]byte, error) {
	return c.doRequest("GET", path, nil)
}

func (c *Client) put(path string, body any) ([]byte, error) {
	return c.doRequestWithRetry("PUT", path, body)
}

func (c *Client) post(path string, body any) ([]byte, error) {
	return c.doRequestWithRetry("POST", path, body)
}

func (c *Client) delete(path string) ([]byte, error) {
	return c.doRequestWithRetry("DELETE", path, nil)
}

type Project struct {
	ID              int64  `json:"id"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	ParentProjectID int64  `json:"parent_project_id"`
	IsArchived      bool   `json:"is_archived"`
	IsFavorite      bool   `json:"is_favorite"`
	Position        int64  `json:"position"`
	HexColor        string `json:"hex_color"`
	Owner           *User  `json:"owner,omitempty"`
	Created         string `json:"created"`
	Updated         string `json:"updated"`
}

func (c *Client) ListProjects() ([]Project, error) {
	body, err := c.get("/api/v1/projects")
	if err != nil {
		return nil, err
	}
	var projects []Project
	if err := json.Unmarshal(body, &projects); err != nil {
		return nil, fmt.Errorf("unmarshal projects: %w", err)
	}
	return projects, nil
}

func (c *Client) GetProject(id int64) (*Project, error) {
	body, err := c.get("/api/v1/projects/" + strconv.FormatInt(id, 10))
	if err != nil {
		return nil, err
	}
	var project Project
	if err := json.Unmarshal(body, &project); err != nil {
		return nil, fmt.Errorf("unmarshal project: %w", err)
	}
	return &project, nil
}

type CreateProjectParams struct {
	Title           string `json:"title"`
	ParentProjectID int64  `json:"parent_project_id,omitempty"`
	Description     string `json:"description,omitempty"`
	HexColor        string `json:"hex_color,omitempty"`
}

func (c *Client) CreateProject(params CreateProjectParams) (*Project, error) {
	body, err := c.put("/api/v1/projects", params)
	if err != nil {
		return nil, err
	}
	var project Project
	if err := json.Unmarshal(body, &project); err != nil {
		return nil, fmt.Errorf("unmarshal project: %w", err)
	}
	return &project, nil
}

type UpdateProjectParams struct {
	Title           string `json:"title,omitempty"`
	Description     string `json:"description,omitempty"`
	IsArchived      *bool  `json:"is_archived,omitempty"`
	ParentProjectID *int64 `json:"parent_project_id,omitempty"`
	Position        *int64 `json:"position,omitempty"`
	HexColor        string `json:"hex_color,omitempty"`
}

func (c *Client) UpdateProject(id int64, params UpdateProjectParams) (*Project, error) {
	body, err := c.post("/api/v1/projects/"+strconv.FormatInt(id, 10), params)
	if err != nil {
		return nil, err
	}
	var project Project
	if err := json.Unmarshal(body, &project); err != nil {
		return nil, fmt.Errorf("unmarshal project: %w", err)
	}
	return &project, nil
}

func (c *Client) DeleteProject(id int64) error {
	_, err := c.delete("/api/v1/projects/" + strconv.FormatInt(id, 10))
	return err
}

type Task struct {
	ID          int64    `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Done        bool     `json:"done"`
	DoneAt      *string  `json:"done_at"`
	DueDate     string   `json:"due_date"`
	Priority    int64    `json:"priority"`
	ProjectID   int64    `json:"project_id"`
	Labels      []Label  `json:"labels"`
	Assignees   []User   `json:"assignees"`
	Created     string   `json:"created"`
	Updated     string   `json:"updated"`
}

type User struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type Label struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	HexColor    string `json:"hex_color"`
}

func (c *Client) ListTasks(projectID *int64) ([]Task, error) {
	path := "/api/v1/tasks"
	if projectID != nil {
		path += "?filter=project_id%3D" + strconv.FormatInt(*projectID, 10)
	}
	body, err := c.get(path)
	if err != nil {
		return nil, err
	}
	var tasks []Task
	if err := json.Unmarshal(body, &tasks); err != nil {
		return nil, fmt.Errorf("unmarshal tasks: %w", err)
	}
	return tasks, nil
}

func (c *Client) GetTask(id int64) (*Task, error) {
	body, err := c.get("/api/v1/tasks/" + strconv.FormatInt(id, 10))
	if err != nil {
		return nil, err
	}
	var task Task
	if err := json.Unmarshal(body, &task); err != nil {
		return nil, fmt.Errorf("unmarshal task: %w", err)
	}
	return &task, nil
}

type LabelRef struct {
	ID int64 `json:"id"`
}

type CreateTaskParams struct {
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	DueDate     string     `json:"due_date,omitempty"`
	Priority    *int64     `json:"priority,omitempty"`
	Labels      []LabelRef `json:"labels,omitempty"`
}

func (c *Client) CreateTask(projectID int64, params CreateTaskParams) (*Task, error) {
	body, err := c.put("/api/v1/projects/"+strconv.FormatInt(projectID, 10)+"/tasks", params)
	if err != nil {
		return nil, err
	}
	var task Task
	if err := json.Unmarshal(body, &task); err != nil {
		return nil, fmt.Errorf("unmarshal task: %w", err)
	}
	return &task, nil
}

type UpdateTaskParams struct {
	Title       string     `json:"title,omitempty"`
	Description string     `json:"description,omitempty"`
	Done        *bool      `json:"done,omitempty"`
	DueDate     string     `json:"due_date,omitempty"`
	Priority    *int64     `json:"priority,omitempty"`
	ProjectID   *int64     `json:"project_id,omitempty"`
	Labels      []LabelRef `json:"labels,omitempty"`
}

func (c *Client) UpdateTask(id int64, params UpdateTaskParams) (*Task, error) {
	body, err := c.post("/api/v1/tasks/"+strconv.FormatInt(id, 10), params)
	if err != nil {
		return nil, err
	}
	var task Task
	if err := json.Unmarshal(body, &task); err != nil {
		return nil, fmt.Errorf("unmarshal task: %w", err)
	}
	return &task, nil
}

func (c *Client) DeleteTask(id int64) error {
	_, err := c.delete("/api/v1/tasks/" + strconv.FormatInt(id, 10))
	return err
}

func (c *Client) ListLabels() ([]Label, error) {
	body, err := c.get("/api/v1/labels")
	if err != nil {
		return nil, err
	}
	var labels []Label
	if err := json.Unmarshal(body, &labels); err != nil {
		return nil, fmt.Errorf("unmarshal labels: %w", err)
	}
	return labels, nil
}

type CreateLabelParams struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	HexColor    string `json:"hex_color,omitempty"`
}

func (c *Client) CreateLabel(params CreateLabelParams) (*Label, error) {
	body, err := c.put("/api/v1/labels", params)
	if err != nil {
		return nil, err
	}
	var label Label
	if err := json.Unmarshal(body, &label); err != nil {
		return nil, fmt.Errorf("unmarshal label: %w", err)
	}
	return &label, nil
}

type Comment struct {
	ID        int64  `json:"id"`
	Comment   string `json:"comment"`
	Author    User   `json:"author"`
	Created   string `json:"created"`
	Updated   string `json:"updated"`
}

func (c *Client) ListTaskComments(taskID int64) ([]Comment, error) {
	body, err := c.get("/api/v1/tasks/" + strconv.FormatInt(taskID, 10) + "/comments")
	if err != nil {
		return nil, err
	}
	var comments []Comment
	if err := json.Unmarshal(body, &comments); err != nil {
		return nil, fmt.Errorf("unmarshal comments: %w", err)
	}
	return comments, nil
}

type CreateCommentParams struct {
	Comment string `json:"comment"`
}

func (c *Client) CreateTaskComment(taskID int64, comment string) (*Comment, error) {
	body, err := c.put("/api/v1/tasks/"+strconv.FormatInt(taskID, 10)+"/comments", CreateCommentParams{Comment: comment})
	if err != nil {
		return nil, err
	}
	var cmt Comment
	if err := json.Unmarshal(body, &cmt); err != nil {
		return nil, fmt.Errorf("unmarshal comment: %w", err)
	}
	return &cmt, nil
}

func (c *Client) ListTaskAssignees(taskID int64) ([]User, error) {
	body, err := c.get("/api/v1/tasks/" + strconv.FormatInt(taskID, 10) + "/assignees")
	if err != nil {
		return nil, err
	}
	var users []User
	if err := json.Unmarshal(body, &users); err != nil {
		return nil, fmt.Errorf("unmarshal assignees: %w", err)
	}
	return users, nil
}

type AddTaskAssigneeParams struct {
	UserID int64 `json:"user_id"`
}

func (c *Client) AddTaskAssignee(taskID, userID int64) error {
	_, err := c.put("/api/v1/tasks/"+strconv.FormatInt(taskID, 10)+"/assignees", AddTaskAssigneeParams{UserID: userID})
	return err
}

func (c *Client) RemoveTaskAssignee(taskID, userID int64) error {
	_, err := c.delete("/api/v1/tasks/" + strconv.FormatInt(taskID, 10) + "/assignees/" + strconv.FormatInt(userID, 10))
	return err
}

type ProjectView struct {
	ID                  int64  `json:"id"`
	Title               string `json:"title"`
	ProjectID           int64  `json:"project_id"`
	ViewKind            string `json:"view_kind"`
	DefaultBucketID     int64  `json:"default_bucket_id"`
	DoneBucketID        int64  `json:"done_bucket_id"`
	BucketConfiguration string `json:"bucket_configuration_mode"`
	Position            float64 `json:"position"`
}

func (c *Client) ListProjectViews(projectID int64) ([]ProjectView, error) {
	body, err := c.get("/api/v1/projects/" + strconv.FormatInt(projectID, 10) + "/views")
	if err != nil {
		return nil, err
	}
	var views []ProjectView
	if err := json.Unmarshal(body, &views); err != nil {
		return nil, fmt.Errorf("unmarshal views: %w", err)
	}
	return views, nil
}

type Bucket struct {
	ID            int64  `json:"id"`
	Title         string `json:"title"`
	ProjectViewID int64  `json:"project_view_id"`
	Count         int64  `json:"count"`
	Position      float64 `json:"position"`
	Limit         int64  `json:"limit"`
}

func (c *Client) ListBuckets(projectID, viewID int64) ([]Bucket, error) {
	body, err := c.get("/api/v1/projects/" + strconv.FormatInt(projectID, 10) + "/views/" + strconv.FormatInt(viewID, 10) + "/buckets")
	if err != nil {
		return nil, err
	}
	var buckets []Bucket
	if err := json.Unmarshal(body, &buckets); err != nil {
		return nil, fmt.Errorf("unmarshal buckets: %w", err)
	}
	return buckets, nil
}

type TaskBucketParams struct {
	TaskID int64 `json:"task_id"`
}

func (c *Client) MoveTaskToBucket(projectID, viewID, bucketID, taskID int64) error {
	_, err := c.post(
		"/api/v1/projects/"+strconv.FormatInt(projectID, 10)+
			"/views/"+strconv.FormatInt(viewID, 10)+
			"/buckets/"+strconv.FormatInt(bucketID, 10)+"/tasks",
		TaskBucketParams{TaskID: taskID},
	)
	return err
}

func (c *Client) AddTaskLabel(taskID, labelID int64) error {
	_, err := c.put("/api/v1/tasks/"+strconv.FormatInt(taskID, 10)+"/labels", LabelRef{ID: labelID})
	return err
}

func (c *Client) RemoveTaskLabel(taskID, labelID int64) error {
	_, err := c.delete("/api/v1/tasks/" + strconv.FormatInt(taskID, 10) + "/labels/" + strconv.FormatInt(labelID, 10))
	return err
}

type BulkLabelsParams struct {
	Labels []LabelRef `json:"labels"`
}

func (c *Client) BulkUpdateTaskLabels(taskID int64, labelIDs []int64) error {
	refs := make([]LabelRef, len(labelIDs))
	for i, id := range labelIDs {
		refs[i] = LabelRef{ID: id}
	}
	_, err := c.post("/api/v1/tasks/"+strconv.FormatInt(taskID, 10)+"/labels/bulk", BulkLabelsParams{Labels: refs})
	return err
}
