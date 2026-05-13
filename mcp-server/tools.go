package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type toolDeps struct {
	client *Client
}

func registerTools(s *server.MCPServer, deps *toolDeps) {
	s.AddTool(listProjectsTool(), deps.handleListProjects)
	s.AddTool(getProjectTool(), deps.handleGetProject)
	s.AddTool(createProjectTool(), deps.handleCreateProject)
	s.AddTool(updateProjectTool(), deps.handleUpdateProject)
	s.AddTool(deleteProjectTool(), deps.handleDeleteProject)
	s.AddTool(listTasksTool(), deps.handleListTasks)
	s.AddTool(getTaskTool(), deps.handleGetTask)
	s.AddTool(createTaskTool(), deps.handleCreateTask)
	s.AddTool(updateTaskTool(), deps.handleUpdateTask)
	s.AddTool(deleteTaskTool(), deps.handleDeleteTask)
	s.AddTool(listTaskCommentsTool(), deps.handleListTaskComments)
	s.AddTool(addTaskCommentTool(), deps.handleAddTaskComment)
	s.AddTool(listTaskAssigneesTool(), deps.handleListTaskAssignees)
	s.AddTool(addTaskAssigneeTool(), deps.handleAddTaskAssignee)
	s.AddTool(removeTaskAssigneeTool(), deps.handleRemoveTaskAssignee)
	s.AddTool(listLabelsTool(), deps.handleListLabels)
	s.AddTool(createLabelTool(), deps.handleCreateLabel)
}

func toolError(format string, args ...any) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultError(fmt.Sprintf(format, args...)), nil
}

func toolResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return toolError("marshal result: %v", err)
	}
	return mcp.NewToolResultText(string(data)), nil
}

func listProjectsTool() mcp.Tool {
	return mcp.NewTool("list_projects",
		mcp.WithDescription("List all projects. Projects can be nested hierarchically using parent_project_id."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
}

func (d *toolDeps) handleListProjects(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projects, err := d.client.ListProjects()
	if err != nil {
		return toolError("list projects: %v", err)
	}
	return toolResult(projects)
}

func getProjectTool() mcp.Tool {
	return mcp.NewTool("get_project",
		mcp.WithDescription("Get details of a specific project by ID."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithInteger("id", mcp.Required(), mcp.Description("The project ID")),
	)
}

func (d *toolDeps) handleGetProject(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireInt("id")
	if err != nil {
		return toolError("id: %v", err)
	}
	project, err := d.client.GetProject(int64(id))
	if err != nil {
		return toolError("get project: %v", err)
	}
	return toolResult(project)
}

func createProjectTool() mcp.Tool {
	return mcp.NewTool("create_project",
		mcp.WithDescription("Create a new project. Optionally nest it under a parent project."),
		mcp.WithString("title", mcp.Required(), mcp.Description("Project title")),
		mcp.WithInteger("parent_project_id", mcp.Description("Parent project ID for nesting (optional, creates a top-level project if omitted)")),
		mcp.WithString("description", mcp.Description("Project description")),
		mcp.WithString("hex_color", mcp.Description("Project color as hex code (e.g. #FF0000)")),
	)
}

func (d *toolDeps) handleCreateProject(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	title, err := req.RequireString("title")
	if err != nil {
		return toolError("title: %v", err)
	}

	params := CreateProjectParams{
		Title:       title,
		Description: req.GetString("description", ""),
		HexColor:    req.GetString("hex_color", ""),
	}
	if v := req.GetInt("parent_project_id", 0); v > 0 {
		pid := int64(v)
		params.ParentProjectID = pid
	}

	project, err := d.client.CreateProject(params)
	if err != nil {
		return toolError("create project: %v", err)
	}
	return toolResult(project)
}

func updateProjectTool() mcp.Tool {
	return mcp.NewTool("update_project",
		mcp.WithDescription("Update an existing project's properties."),
		mcp.WithInteger("id", mcp.Required(), mcp.Description("The project ID to update")),
		mcp.WithString("title", mcp.Description("New project title")),
		mcp.WithString("description", mcp.Description("New project description")),
		mcp.WithBoolean("is_archived", mcp.Description("Whether the project is archived")),
	)
}

func (d *toolDeps) handleUpdateProject(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireInt("id")
	if err != nil {
		return toolError("id: %v", err)
	}

	params := UpdateProjectParams{}
	if v, ok := req.GetArguments()["title"]; ok {
		params.Title = v.(string)
	}
	if v, ok := req.GetArguments()["description"]; ok {
		params.Description = v.(string)
	}
	if v, ok := req.GetArguments()["is_archived"]; ok {
		b := v.(bool)
		params.IsArchived = &b
	}

	project, err := d.client.UpdateProject(int64(id), params)
	if err != nil {
		return toolError("update project: %v", err)
	}
	return toolResult(project)
}

func deleteProjectTool() mcp.Tool {
	return mcp.NewTool("delete_project",
		mcp.WithDescription("Delete a project by ID. This will also delete all tasks in the project."),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithInteger("id", mcp.Required(), mcp.Description("The project ID to delete")),
	)
}

func (d *toolDeps) handleDeleteProject(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireInt("id")
	if err != nil {
		return toolError("id: %v", err)
	}
	if err := d.client.DeleteProject(int64(id)); err != nil {
		return toolError("delete project: %v", err)
	}
	return mcp.NewToolResultText(fmt.Sprintf("Project %d deleted successfully", id)), nil
}

func listTasksTool() mcp.Tool {
	return mcp.NewTool("list_tasks",
		mcp.WithDescription("List tasks, optionally filtered by project ID."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithInteger("project_id", mcp.Description("Filter tasks by project ID")),
	)
}

func (d *toolDeps) handleListTasks(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var projectID *int64
	if v, ok := req.GetArguments()["project_id"]; ok {
		pid := int64(0)
		switch val := v.(type) {
		case int:
			pid = int64(val)
		case float64:
			pid = int64(val)
		}
		if pid > 0 {
			projectID = &pid
		}
	}
	tasks, err := d.client.ListTasks(projectID)
	if err != nil {
		return toolError("list tasks: %v", err)
	}
	return toolResult(tasks)
}

func getTaskTool() mcp.Tool {
	return mcp.NewTool("get_task",
		mcp.WithDescription("Get full details of a specific task by ID."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithInteger("id", mcp.Required(), mcp.Description("The task ID")),
	)
}

func (d *toolDeps) handleGetTask(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireInt("id")
	if err != nil {
		return toolError("id: %v", err)
	}
	task, err := d.client.GetTask(int64(id))
	if err != nil {
		return toolError("get task: %v", err)
	}
	return toolResult(task)
}

func createTaskTool() mcp.Tool {
	return mcp.NewTool("create_task",
		mcp.WithDescription("Create a new task in a project."),
		mcp.WithInteger("project_id", mcp.Required(), mcp.Description("The project ID to create the task in")),
		mcp.WithString("title", mcp.Required(), mcp.Description("Task title")),
		mcp.WithString("description", mcp.Description("Task description in markdown")),
		mcp.WithString("due_date", mcp.Description("Due date in ISO 8601 format (e.g. 2025-12-31T00:00:00Z)")),
		mcp.WithInteger("priority", mcp.Description("Task priority (1=urgent, 2=high, 3=medium, 4=low, 5=none)")),
	)
}

func (d *toolDeps) handleCreateTask(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireInt("project_id")
	if err != nil {
		return toolError("project_id: %v", err)
	}
	title, err := req.RequireString("title")
	if err != nil {
		return toolError("title: %v", err)
	}

	params := CreateTaskParams{
		Title:       title,
		Description: req.GetString("description", ""),
		DueDate:     req.GetString("due_date", ""),
	}

	if v, ok := req.GetArguments()["priority"]; ok {
		switch val := v.(type) {
		case int:
			p := int64(val)
			params.Priority = &p
		case float64:
			p := int64(val)
			params.Priority = &p
		}
	}

	task, err := d.client.CreateTask(int64(projectID), params)
	if err != nil {
		return toolError("create task: %v", err)
	}
	return toolResult(task)
}

func updateTaskTool() mcp.Tool {
	return mcp.NewTool("update_task",
		mcp.WithDescription("Update an existing task's properties. Use this to mark tasks as done, change titles, reassign projects, etc."),
		mcp.WithInteger("id", mcp.Required(), mcp.Description("The task ID to update")),
		mcp.WithString("title", mcp.Description("New task title")),
		mcp.WithString("description", mcp.Description("New task description")),
		mcp.WithBoolean("done", mcp.Description("Mark the task as done or not done")),
		mcp.WithString("due_date", mcp.Description("New due date in ISO 8601 format")),
		mcp.WithInteger("priority", mcp.Description("New priority (1=urgent, 2=high, 3=medium, 4=low, 5=none)")),
		mcp.WithInteger("project_id", mcp.Description("Move task to a different project")),
	)
}

func (d *toolDeps) handleUpdateTask(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireInt("id")
	if err != nil {
		return toolError("id: %v", err)
	}

	params := UpdateTaskParams{}
	if v, ok := req.GetArguments()["title"]; ok {
		params.Title = v.(string)
	}
	if v, ok := req.GetArguments()["description"]; ok {
		params.Description = v.(string)
	}
	if v, ok := req.GetArguments()["done"]; ok {
		b := v.(bool)
		params.Done = &b
	}
	if v, ok := req.GetArguments()["due_date"]; ok {
		params.DueDate = v.(string)
	}
	if v, ok := req.GetArguments()["priority"]; ok {
		switch val := v.(type) {
		case int:
			p := int64(val)
			params.Priority = &p
		case float64:
			p := int64(val)
			params.Priority = &p
		}
	}
	if v, ok := req.GetArguments()["project_id"]; ok {
		switch val := v.(type) {
		case int:
			p := int64(val)
			params.ProjectID = &p
		case float64:
			p := int64(val)
			params.ProjectID = &p
		}
	}

	task, err := d.client.UpdateTask(int64(id), params)
	if err != nil {
		return toolError("update task: %v", err)
	}
	return toolResult(task)
}

func deleteTaskTool() mcp.Tool {
	return mcp.NewTool("delete_task",
		mcp.WithDescription("Delete a task by ID."),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithInteger("id", mcp.Required(), mcp.Description("The task ID to delete")),
	)
}

func (d *toolDeps) handleDeleteTask(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireInt("id")
	if err != nil {
		return toolError("id: %v", err)
	}
	if err := d.client.DeleteTask(int64(id)); err != nil {
		return toolError("delete task: %v", err)
	}
	return mcp.NewToolResultText(fmt.Sprintf("Task %d deleted successfully", id)), nil
}

func listTaskCommentsTool() mcp.Tool {
	return mcp.NewTool("list_task_comments",
		mcp.WithDescription("List all comments on a task."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithInteger("task_id", mcp.Required(), mcp.Description("The task ID")),
	)
}

func (d *toolDeps) handleListTaskComments(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	taskID, err := req.RequireInt("task_id")
	if err != nil {
		return toolError("task_id: %v", err)
	}
	comments, err := d.client.ListTaskComments(int64(taskID))
	if err != nil {
		return toolError("list comments: %v", err)
	}
	return toolResult(comments)
}

func addTaskCommentTool() mcp.Tool {
	return mcp.NewTool("add_task_comment",
		mcp.WithDescription("Add a comment to a task."),
		mcp.WithInteger("task_id", mcp.Required(), mcp.Description("The task ID")),
		mcp.WithString("comment", mcp.Required(), mcp.Description("The comment text in markdown")),
	)
}

func (d *toolDeps) handleAddTaskComment(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	taskID, err := req.RequireInt("task_id")
	if err != nil {
		return toolError("task_id: %v", err)
	}
	comment, err := req.RequireString("comment")
	if err != nil {
		return toolError("comment: %v", err)
	}
	cmt, err := d.client.CreateTaskComment(int64(taskID), comment)
	if err != nil {
		return toolError("add comment: %v", err)
	}
	return toolResult(cmt)
}

func listLabelsTool() mcp.Tool {
	return mcp.NewTool("list_labels",
		mcp.WithDescription("List all available labels."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
}

func (d *toolDeps) handleListLabels(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	labels, err := d.client.ListLabels()
	if err != nil {
		return toolError("list labels: %v", err)
	}
	return toolResult(labels)
}

func createLabelTool() mcp.Tool {
	return mcp.NewTool("create_label",
		mcp.WithDescription("Create a new label."),
		mcp.WithString("title", mcp.Required(), mcp.Description("Label title")),
		mcp.WithString("description", mcp.Description("Label description")),
		mcp.WithString("hex_color", mcp.Description("Label color as hex code (e.g. #FF0000)")),
	)
}

func (d *toolDeps) handleCreateLabel(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	title, err := req.RequireString("title")
	if err != nil {
		return toolError("title: %v", err)
	}

	params := CreateLabelParams{
		Title:       title,
		Description: req.GetString("description", ""),
		HexColor:    req.GetString("hex_color", ""),
	}

	label, err := d.client.CreateLabel(params)
	if err != nil {
		return toolError("create label: %v", err)
	}
	return toolResult(label)
}

func listTaskAssigneesTool() mcp.Tool {
	return mcp.NewTool("list_task_assignees",
		mcp.WithDescription("List all users assigned to a task."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithInteger("task_id", mcp.Required(), mcp.Description("The task ID")),
	)
}

func (d *toolDeps) handleListTaskAssignees(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	taskID, err := req.RequireInt("task_id")
	if err != nil {
		return toolError("task_id: %v", err)
	}
	users, err := d.client.ListTaskAssignees(int64(taskID))
	if err != nil {
		return toolError("list assignees: %v", err)
	}
	return toolResult(users)
}

func addTaskAssigneeTool() mcp.Tool {
	return mcp.NewTool("add_task_assignee",
		mcp.WithDescription("Assign a user to a task. The user needs access to the project."),
		mcp.WithInteger("task_id", mcp.Required(), mcp.Description("The task ID")),
		mcp.WithInteger("user_id", mcp.Required(), mcp.Description("The user ID to assign")),
	)
}

func (d *toolDeps) handleAddTaskAssignee(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	taskID, err := req.RequireInt("task_id")
	if err != nil {
		return toolError("task_id: %v", err)
	}
	userID, err := req.RequireInt("user_id")
	if err != nil {
		return toolError("user_id: %v", err)
	}
	if err := d.client.AddTaskAssignee(int64(taskID), int64(userID)); err != nil {
		return toolError("add assignee: %v", err)
	}
	return mcp.NewToolResultText(fmt.Sprintf("User %d assigned to task %d", userID, taskID)), nil
}

func removeTaskAssigneeTool() mcp.Tool {
	return mcp.NewTool("remove_task_assignee",
		mcp.WithDescription("Remove a user assignment from a task."),
		mcp.WithInteger("task_id", mcp.Required(), mcp.Description("The task ID")),
		mcp.WithInteger("user_id", mcp.Required(), mcp.Description("The user ID to unassign")),
	)
}

func (d *toolDeps) handleRemoveTaskAssignee(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	taskID, err := req.RequireInt("task_id")
	if err != nil {
		return toolError("task_id: %v", err)
	}
	userID, err := req.RequireInt("user_id")
	if err != nil {
		return toolError("user_id: %v", err)
	}
	if err := d.client.RemoveTaskAssignee(int64(taskID), int64(userID)); err != nil {
		return toolError("remove assignee: %v", err)
	}
	return mcp.NewToolResultText(fmt.Sprintf("User %d unassigned from task %d", userID, taskID)), nil
}

