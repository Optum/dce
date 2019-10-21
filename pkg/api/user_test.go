package api

import (
	"fmt"
	"testing"

	"github.com/Optum/Redbox/pkg/awsiface/mocks"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/stretchr/testify/require"
)

func TestUser(t *testing.T) {

	t.Run("NonCognitoAuthIsAdmin, Output", func(t *testing.T) {

		mockCognitoIdp := &mocks.CognitoIdentityProviderAPI{}
		userGetter := UserDetails{
			CognitoUserPoolID:        "us_east_1-test",
			RolesAttributesAdminName: "admin",
			CognitoClient:            mockCognitoIdp,
		}

		user := userGetter.GetUser(&events.APIGatewayProxyRequest{
			RequestContext: events.APIGatewayProxyRequestContext{
				Identity: events.APIGatewayRequestIdentity{
					CognitoIdentityPoolID: "",
				},
			},
		})
		require.Equal(t, user.userName, "")
		require.Equal(t, user.role, AdminGroupName)
	})

	t.Run("CognitoAuthInAdminsGroup, Output", func(t *testing.T) {

		mockCognitoIdp := &mocks.CognitoIdentityProviderAPI{}
		userGetter := UserDetails{
			CognitoUserPoolID:        "us_east_1-test",
			RolesAttributesAdminName: "admins",
			CognitoClient:            mockCognitoIdp,
		}

		mockCognitoIdp.On("ListUsers", &cognitoidentityprovider.ListUsersInput{
			Filter:     aws.String("sub = \"abcdef-123456\""),
			UserPoolId: aws.String("us_east_1-test"),
		}).Return(&cognitoidentityprovider.ListUsersOutput{
			Users: []*cognitoidentityprovider.UserType{
				{
					Username: aws.String("testuser"),
				},
			},
		}, nil)
		mockCognitoIdp.On("AdminListGroupsForUser", &cognitoidentityprovider.AdminListGroupsForUserInput{
			Username:   aws.String("testuser"),
			UserPoolId: aws.String("us_east_1-test"),
		}).Return(&cognitoidentityprovider.AdminListGroupsForUserOutput{
			Groups: []*cognitoidentityprovider.GroupType{
				{
					GroupName: aws.String("Admins"),
				},
			},
		}, nil)

		user := userGetter.GetUser(&events.APIGatewayProxyRequest{
			RequestContext: events.APIGatewayProxyRequestContext{
				Identity: events.APIGatewayRequestIdentity{
					CognitoIdentityPoolID:         "us_east_1-test",
					CognitoAuthenticationProvider: "UserPoolID:CognitoSignIn:abcdef-123456",
				},
			},
		})
		require.Equal(t, user.userName, "testuser")
		require.Equal(t, user.role, AdminGroupName)
	})

	t.Run("CognitoAuthInAdminsRoleAttributes, Output", func(t *testing.T) {

		mockCognitoIdp := &mocks.CognitoIdentityProviderAPI{}
		userGetter := UserDetails{
			CognitoUserPoolID:        "us_east_1-test",
			RolesAttributesAdminName: "admins",
			CognitoClient:            mockCognitoIdp,
		}

		mockCognitoIdp.On("ListUsers", &cognitoidentityprovider.ListUsersInput{
			Filter:     aws.String("sub = \"abcdef-123456\""),
			UserPoolId: aws.String("us_east_1-test"),
		}).Return(&cognitoidentityprovider.ListUsersOutput{
			Users: []*cognitoidentityprovider.UserType{
				{
					Username: aws.String("testuser"),
					Attributes: []*cognitoidentityprovider.AttributeType{
						{
							Name:  aws.String("custom:roles"),
							Value: aws.String("group1, group2,group3, admins"),
						},
					},
				},
			},
		}, nil)
		mockCognitoIdp.On("AdminListGroupsForUser", &cognitoidentityprovider.AdminListGroupsForUserInput{
			Username:   aws.String("testuser"),
			UserPoolId: aws.String("us_east_1-test"),
		}).Return(&cognitoidentityprovider.AdminListGroupsForUserOutput{
			Groups: []*cognitoidentityprovider.GroupType{},
		}, nil)

		user := userGetter.GetUser(&events.APIGatewayProxyRequest{
			RequestContext: events.APIGatewayProxyRequestContext{
				Identity: events.APIGatewayRequestIdentity{
					CognitoIdentityPoolID:         "us_east_1-test",
					CognitoAuthenticationProvider: "UserPoolID:CognitoSignIn:abcdef-123456",
				},
			},
		})
		require.Equal(t, user.userName, "testuser")
		require.Equal(t, user.role, AdminGroupName)
	})

	t.Run("LookForStringInCommaList, Output", func(t *testing.T) {

		mockCognitoIdp := &mocks.CognitoIdentityProviderAPI{}
		userGetter := UserDetails{
			CognitoUserPoolID:        "us_east_1-test",
			RolesAttributesAdminName: "admins",
			CognitoClient:            mockCognitoIdp,
		}

		require.True(t, userGetter.isUserInAdminFromList(aws.String("Group1,Group2,admins")))
		require.True(t, userGetter.isUserInAdminFromList(aws.String("admins")))
		require.False(t, userGetter.isUserInAdminFromList(aws.String("Group1,Group2,")))
		require.False(t, userGetter.isUserInAdminFromList(aws.String("Admin")))
	})

	t.Run("CognitoAuthInAdminsGroup, Output", func(t *testing.T) {

		mockCognitoIdp := &mocks.CognitoIdentityProviderAPI{}
		userGetter := UserDetails{
			CognitoUserPoolID:        "us_east_1-test",
			RolesAttributesAdminName: "admins",
			CognitoClient:            mockCognitoIdp,
		}

		mockCognitoIdp.On("AdminListGroupsForUser", &cognitoidentityprovider.AdminListGroupsForUserInput{
			Username:   aws.String("testuser"),
			UserPoolId: aws.String("us_east_1-test"),
		}).Return(&cognitoidentityprovider.AdminListGroupsForUserOutput{
			Groups: []*cognitoidentityprovider.GroupType{
				{
					GroupName: aws.String("Admins"),
				},
			},
		}, nil)

		userIsAdmin, err := userGetter.isUserInAdminGroup("testuser")
		require.Nil(t, err)
		require.True(t, userIsAdmin)
	})
	t.Run("CognitoAuthNotInAdminsGroup, Output", func(t *testing.T) {

		mockCognitoIdp := &mocks.CognitoIdentityProviderAPI{}
		userGetter := UserDetails{
			CognitoUserPoolID:        "us_east_1-test",
			RolesAttributesAdminName: "admins",
			CognitoClient:            mockCognitoIdp,
		}

		mockCognitoIdp.On("AdminListGroupsForUser", &cognitoidentityprovider.AdminListGroupsForUserInput{
			Username:   aws.String("testuser"),
			UserPoolId: aws.String("us_east_1-test"),
		}).Return(&cognitoidentityprovider.AdminListGroupsForUserOutput{
			Groups: []*cognitoidentityprovider.GroupType{
				{
					GroupName: aws.String("Users"),
				},
			},
		}, nil)

		userIsAdmin, err := userGetter.isUserInAdminGroup("testuser")
		require.Nil(t, err)
		require.False(t, userIsAdmin)
	})
	t.Run("CognitoAuthInAdminsGroupMultiple, Output", func(t *testing.T) {

		mockCognitoIdp := &mocks.CognitoIdentityProviderAPI{}
		userGetter := UserDetails{
			CognitoUserPoolID:        "us_east_1-test",
			RolesAttributesAdminName: "admins",
			CognitoClient:            mockCognitoIdp,
		}

		mockCognitoIdp.On("AdminListGroupsForUser", &cognitoidentityprovider.AdminListGroupsForUserInput{
			Username:   aws.String("testuser"),
			UserPoolId: aws.String("us_east_1-test"),
		}).Return(&cognitoidentityprovider.AdminListGroupsForUserOutput{
			Groups: []*cognitoidentityprovider.GroupType{
				{
					GroupName: aws.String("Users"),
				},
				{
					GroupName: aws.String("Admins"),
				},
			},
		}, nil)

		userIsAdmin, err := userGetter.isUserInAdminGroup("testuser")
		require.Nil(t, err)
		require.True(t, userIsAdmin)
	})
	t.Run("CognitoAuthInAdminsError, Output", func(t *testing.T) {

		mockCognitoIdp := &mocks.CognitoIdentityProviderAPI{}
		userGetter := UserDetails{
			CognitoUserPoolID:        "us_east_1-test",
			RolesAttributesAdminName: "admins",
			CognitoClient:            mockCognitoIdp,
		}

		mockCognitoIdp.On("AdminListGroupsForUser", &cognitoidentityprovider.AdminListGroupsForUserInput{
			Username:   aws.String("testuser"),
			UserPoolId: aws.String("us_east_1-test"),
		}).Return(&cognitoidentityprovider.AdminListGroupsForUserOutput{}, fmt.Errorf("Fail"))

		userIsAdmin, err := userGetter.isUserInAdminGroup("testuser")
		require.NotNil(t, err)
		require.False(t, userIsAdmin)
	})
}
