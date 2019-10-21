package api

import (
	"fmt"
	"strings"

	"github.com/Optum/Redbox/pkg/awsiface"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
)

// UserGroupName - Has the string to define Users
const UserGroupName = "User"

// AdminGroupName - Has a string to define Admins
const AdminGroupName = "Admin"

// User - Has the username and their role
type User struct {
	userName string
	role     string
}

// UserDetailer - used for mocking tests
type UserDetailer interface {
	GetUser(event *events.APIGatewayProxyRequest) *User
	isUserInAdminGroup(username string) (bool, error)
	isUserInAdminFromList(groups *string) bool
}

// UserDetails - Gets User information
type UserDetails struct {
	CognitoUserPoolID        string
	RolesAttributesAdminName string
	CognitoClient            awsiface.CognitoIdentityProviderAPI
}

// GetUser - Gets the username and role out of an event
func (u *UserDetails) GetUser(event *events.APIGatewayProxyRequest) *User {

	if event.RequestContext.Identity.CognitoIdentityPoolID == "" {
		// No cognito authentication means the user is considered an admin
		return &User{
			role: AdminGroupName,
		}
	}

	congitoSubID := strings.Split(event.RequestContext.Identity.CognitoAuthenticationProvider, ":CognitoSignIn:")[1]

	filter := fmt.Sprintf("sub = \"%s\"", congitoSubID)
	users, err := u.CognitoClient.ListUsers(&cognitoidentityprovider.ListUsersInput{
		Filter:     aws.String(filter),
		UserPoolId: aws.String(u.CognitoUserPoolID),
	})
	if err != nil {
		fmt.Printf("Error listing users from Cognito: %s", err)
		return &User{}
	}
	if len(users.Users) != 1 {
		fmt.Printf("Did not get the current user.  Found %d instead of 1.", len(users.Users))
		return &User{}
	}

	user := &User{
		role:     UserGroupName,
		userName: *users.Users[0].Username,
	}

	for _, attribute := range users.Users[0].Attributes {
		if *attribute.Name == "custom:roles" {
			if u.isUserInAdminFromList(attribute.Value) {
				user.role = AdminGroupName
				return user
			}
		}
	}

	isAdmin, err := u.isUserInAdminGroup(user.userName)
	if err != nil {
		fmt.Printf("Got an error when quering groups for user: %s", err)
		return user
	}
	if isAdmin {
		user.role = AdminGroupName
		return user
	}

	return user
}

func (u *UserDetails) isUserInAdminGroup(username string) (bool, error) {

	groups, err := u.CognitoClient.AdminListGroupsForUser(&cognitoidentityprovider.AdminListGroupsForUserInput{
		Username:   aws.String(username),
		UserPoolId: aws.String(u.CognitoUserPoolID),
	})
	if err != nil {
		fmt.Printf("Was not abile to query a users for its groups: %s", err)
		return false, fmt.Errorf("Was not abile to query a users for its groups: %s", err)
	}
	for _, group := range groups.Groups {
		if *group.GroupName == "Admins" {
			return true, nil
		}
	}
	return false, nil
}

func (u *UserDetails) isUserInAdminFromList(groups *string) bool {

	for _, group := range strings.Split(*groups, ",") {
		if strings.TrimSpace(group) == u.RolesAttributesAdminName {
			return true
		}
	}
	return false
}
