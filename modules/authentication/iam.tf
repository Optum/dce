data "template_file" "user_assume_role_policy" {
  template = "${
    file("${path.module}/fixtures/iam/assume-role.json")
  }"

  vars = {
    cognito_identity_pool_id = "${aws_cognito_identity_pool._.id}"
  }
}

data "template_file" "user_policy" {
  template = "${file("${path.module}/fixtures/iam/user-policy.json")}"

  vars = {
    api_gateway_arn = "${var.api_gateway_arn}"
  }
}

resource "aws_iam_role" "user" {
  name = "${var.name}-user-${var.namespace}"

  assume_role_policy = "${
    data.template_file.user_assume_role_policy.rendered
  }"
}

resource "aws_iam_policy" "user" {
  name = "${var.name}-user-${var.namespace}"

  policy = "${data.template_file.user_policy.rendered}"
}
resource "aws_iam_policy_attachment" "user" {
  name = "${var.name}-user-${var.namespace}"

  policy_arn = "${aws_iam_policy.user.arn}"
  roles      = ["${aws_iam_role.user.name}"]
}


data "template_file" "admin_assume_role_policy" {
  template = "${
    file("${path.module}/fixtures/iam/assume-role.json")
  }"

  vars = {
    cognito_identity_pool_id = "${aws_cognito_identity_pool._.id}"
  }
}

data "template_file" "admin_policy" {
  template = "${file("${path.module}/fixtures/iam/admin-policy.json")}"

  vars = {
    api_gateway_arn = "${var.api_gateway_arn}"
  }
}

resource "aws_iam_role" "admin" {
  name = "${var.name}-admin-${var.namespace}"

  assume_role_policy = "${
    data.template_file.admin_assume_role_policy.rendered
  }"
}

resource "aws_iam_policy" "admin" {
  name = "${var.name}-admin-${var.namespace}"

  policy = "${data.template_file.admin_policy.rendered}"
}
resource "aws_iam_policy_attachment" "admin" {
  name = "${var.name}-admin-${var.namespace}"

  policy_arn = "${aws_iam_policy.admin.arn}"
  roles      = ["${aws_iam_role.admin.name}"]
}
