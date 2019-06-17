# Contribution Guidelines

Please note that this project is released with a [Contributor Code of Conduct](CODE_OF_CONDUCT.md). By participating in this project you agree to abide by its terms. Please also review our [Individual Contributor License Agreement ("ICL")](INDIVIDUAL_CONTRIBUTOR_LICENSE.md) prior to submitting changes to the project.  You will need to attest to this agreement following the instructions below.

---

# How to Contribute

We welcome PRs and will aim to respond to them within 2 business days.

## Coding Standards

We have some general guidelines towards contributing to this project.

### Languages

#### Golang

The Lambda and CodeBuild function code is written in Golang.  We prefer that similar contributed code also be written in Golang.  Please ensure your Golang code is formatted by [gofmt](https://golang.org/cmd/gofmt/) and linted by [golint](https://godoc.org/golang.org/x/lint).

## Pull Requests

All submissions should come in the form of pull requests and need to be reviewed by project committers. Read [GitHub's pull request documentation](https://help.github.com/en/articles/about-pull-requests) for more information on sending pull requests.

### Commit Signing & Individual Contributor License

* Please read this guide and make sure you agree with our [Individual Contributor License Agreement ("ICL")](INDIVIDUAL_CONTRIBUTOR_LICENSE.md).
* Make sure git knows your name and email address:
   ```
   $ git config user.name "J. Random User"
   $ git config user.email "j.random.user@example.com"
   ```
> Signing-Off on your commit is agreeing with our ICL and attesting that you either wrote or have the rights to contribute the code. The name and email address must be valid as we cannot accept anonymous contributions.
* Write good commit messages
* Sign-off every commit `git commit --signoff` or `git commit -s`, as directed by the ICL.

> If you forget to sign a commit, then you’ll have to do a bit of rewriting history. Don’t worry. It’s pretty easy. If it’s the latest commit, you can just run either `git commit -a -s` or `git commit --amend --signoff` to fix things. It gets a little trickier if you have to change something farther back in history but there are some good instructions [here](https://git-scm.com/book/en/v2/Git-Tools-Rewriting-History) in the Changing Multiple Commit Messages section.

### Branch/Fork Mechanics

1. Fork this repo
1. Clone your forked repo
1. Implement desired changes in `master` branch
1. Validate the changes meet your desired use case
1. Validate that existing tests pass
1. Write additional tests to validate your added/changed functionality
1. Update documentation
1. Push to your forked repo
1. Open a pull request and complete the provided PR template. Thank you for your contribution! A dialog will ensue.

### Updating Your Pull Request

Many times, making a PR adhere to the standards above can be difficult. If the maintainers notice anything that we'd like changed, we'll ask you to edit your PR before we merge it. This applies to both the content documented in the PR and the code change itself.

[email]: mailto:opensource@optum.com
