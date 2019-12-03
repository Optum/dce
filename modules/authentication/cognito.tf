resource "aws_cognito_user_pool" "_" {
  name = "${var.name}${var.namespace}"

  admin_create_user_config {
    allow_admin_create_user_only = true
  }

  // Add a custom attribute we can use to determine
  // who an admin is
  schema {
    attribute_data_type      = "String"
    developer_only_attribute = false
    mutable                  = true
    name                     = "roles"
    required                 = false

    string_attribute_constraints {}
  }
}

resource "aws_cognito_user_pool_client" "_" {
  name = "${var.name}${var.namespace}"

  user_pool_id                         = aws_cognito_user_pool._.id
  generate_secret                      = false
  allowed_oauth_flows                  = ["implicit", "code"]
  allowed_oauth_scopes                 = ["profile", "openid", "email"]
  supported_identity_providers         = var.identity_providers
  allowed_oauth_flows_user_pool_client = true
  callback_urls                        = var.callback_urls
  logout_urls                          = var.logout_urls
}

resource "aws_cognito_user_pool_domain" "_" {
  domain       = "${lower(var.name)}-${data.aws_caller_identity.current.account_id}-${lower(var.namespace)}"
  user_pool_id = aws_cognito_user_pool._.id
}

resource "aws_cognito_identity_pool" "_" {
  identity_pool_name               = replace("${var.name}${var.namespace}", "/-/", " ")
  allow_unauthenticated_identities = false

  cognito_identity_providers {
    client_id               = aws_cognito_user_pool_client._.id
    provider_name           = aws_cognito_user_pool._.endpoint
    server_side_token_check = true
  }
}

resource "aws_cognito_identity_pool_roles_attachment" "_" {
  identity_pool_id = aws_cognito_identity_pool._.id

  role_mapping {
    identity_provider         = "${aws_cognito_user_pool._.endpoint}:${aws_cognito_user_pool_client._.id}"
    ambiguous_role_resolution = "AuthenticatedRole"
    type                      = "Rules"

    mapping_rule {
      claim      = "cognito:groups"
      match_type = "Contains"
      role_arn   = "${aws_iam_role.admin.arn}"
      value      = "Admin"
    }

    mapping_rule {
      claim      = "custom:roles"
      match_type = "Contains"
      role_arn   = "${aws_iam_role.admin.arn}"
      value      = "Admin"
    }
  }

  roles = {
    "authenticated" = "${aws_iam_role.user.arn}"
  }
}
