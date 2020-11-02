// Code generated by github.com/Yamashou/gqlgenc, DO NOT EDIT.

package gen

import (
	"context"

	"github.com/Yamashou/gqlgenc/client"
)

type Client struct {
	Client *client.HTTPClient
}
type UserFragment struct {
	Name     *string "json:\"name\" graphql:\"name\""
	Username string  "json:\"username\" graphql:\"username\""
}
type GetUsers struct {
	QueryUser []*struct {
		Name     *string "json:\"name\" graphql:\"name\""
		Username string  "json:\"username\" graphql:\"username\""
		Tasks    []*struct {
			Title     string "json:\"title\" graphql:\"title\""
			Completed bool   "json:\"completed\" graphql:\"completed\""
		} "json:\"tasks\" graphql:\"tasks\""
	} "json:\"queryUser\" graphql:\"queryUser\""
}
type AddTasksPayload struct {
	AddTask *struct {
		NumUids *int "json:\"numUids\" graphql:\"numUids\""
		Task    []*struct {
			ID    string       "json:\"id\" graphql:\"id\""
			Title string       "json:\"title\" graphql:\"title\""
			User  UserFragment "json:\"user\" graphql:\"user\""
		} "json:\"task\" graphql:\"task\""
	} "json:\"addTask\" graphql:\"addTask\""
}
type AddUsersMutationPayload struct {
	AddUser *struct {
		NumUids *int            "json:\"numUids\" graphql:\"numUids\""
		User    []*UserFragment "json:\"user\" graphql:\"user\""
	} "json:\"addUser\" graphql:\"addUser\""
}
type DeleteUserMutationPayload struct {
	DeleteUser *struct {
		NumUids *int    "json:\"numUids\" graphql:\"numUids\""
		Msg     *string "json:\"msg\" graphql:\"msg\""
	} "json:\"deleteUser\" graphql:\"deleteUser\""
}
type DeleteTaskMutationPayload struct {
	DeleteTask *struct {
		NumUids *int    "json:\"numUids\" graphql:\"numUids\""
		Msg     *string "json:\"msg\" graphql:\"msg\""
	} "json:\"deleteTask\" graphql:\"deleteTask\""
}

const GetUsersQuery = `query GetUsers {
	queryUser {
		... UserFragment
		tasks {
			title
			completed
		}
	}
}
fragment UserFragment on User {
	name
	username
}
`

func (c *Client) GetUsers(ctx context.Context, httpRequestOptions ...client.HTTPRequestOption) (*GetUsers, error) {
	vars := map[string]interface{}{}

	var res GetUsers
	if err := c.Client.Post(ctx, GetUsersQuery, &res, vars, httpRequestOptions...); err != nil {
		return nil, err
	}

	return &res, nil
}

const AddTasksQuery = `mutation AddTasks ($input: [AddTaskInput!]!) {
	addTask(input: $input) {
		numUids
		task {
			id
			title
			user {
				... UserFragment
			}
		}
	}
}
fragment UserFragment on User {
	name
	username
}
`

func (c *Client) AddTasks(ctx context.Context, input []*AddTaskInput, httpRequestOptions ...client.HTTPRequestOption) (*AddTasksPayload, error) {
	vars := map[string]interface{}{
		"input": input,
	}

	var res AddTasksPayload
	if err := c.Client.Post(ctx, AddTasksQuery, &res, vars, httpRequestOptions...); err != nil {
		return nil, err
	}

	return &res, nil
}

const AddUsersMutationQuery = `mutation AddUsersMutation ($input: [AddUserInput!]!) {
	addUser(input: $input) {
		numUids
		user {
			... UserFragment
		}
	}
}
fragment UserFragment on User {
	name
	username
}
`

func (c *Client) AddUsersMutation(ctx context.Context, input []*AddUserInput, httpRequestOptions ...client.HTTPRequestOption) (*AddUsersMutationPayload, error) {
	vars := map[string]interface{}{
		"input": input,
	}

	var res AddUsersMutationPayload
	if err := c.Client.Post(ctx, AddUsersMutationQuery, &res, vars, httpRequestOptions...); err != nil {
		return nil, err
	}

	return &res, nil
}

const DeleteUserMutationQuery = `mutation DeleteUserMutation ($username: String!) {
	deleteUser(filter: {username:{eq:$username}}) {
		numUids
		msg
	}
}
`

func (c *Client) DeleteUserMutation(ctx context.Context, username string, httpRequestOptions ...client.HTTPRequestOption) (*DeleteUserMutationPayload, error) {
	vars := map[string]interface{}{
		"username": username,
	}

	var res DeleteUserMutationPayload
	if err := c.Client.Post(ctx, DeleteUserMutationQuery, &res, vars, httpRequestOptions...); err != nil {
		return nil, err
	}

	return &res, nil
}

const DeleteTaskMutationQuery = `mutation DeleteTaskMutation ($id: ID!) {
	deleteTask(filter: {id:[$id]}) {
		numUids
		msg
	}
}
`

func (c *Client) DeleteTaskMutation(ctx context.Context, id string, httpRequestOptions ...client.HTTPRequestOption) (*DeleteTaskMutationPayload, error) {
	vars := map[string]interface{}{
		"id": id,
	}

	var res DeleteTaskMutationPayload
	if err := c.Client.Post(ctx, DeleteTaskMutationQuery, &res, vars, httpRequestOptions...); err != nil {
		return nil, err
	}

	return &res, nil
}
