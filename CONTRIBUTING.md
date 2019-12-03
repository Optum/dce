# Contribution Guidelines

Please note that this project is released with a [Contributor Code of Conduct](CODE_OF_CONDUCT.md). By participating in this project you agree to abide by its terms. Please also review our [Individual Contributor License Agreement ("ICL")](INDIVIDUAL_CONTRIBUTOR_LICENSE.md) prior to submitting changes to the project.  You will need to attest to this agreement following the instructions in the [Paperwork for Pull Requests](#paperwork-for-pull-requests) section below.

---

# How to Contribute

Now that we have the disclaimer out of the way, let's get into how you can be a part of our project. There are many different ways to contribute.

## Issues

We track our work using Issues in GitHub. Feel free to open up your own issue to point out areas for improvement or to suggest your own new experiment. If you are comfortable with signing the waiver linked above and contributing code or documentation, grab your own issue and start working.

## Coding Standards

We have some general guidelines towards contributing to this project.

### Languages

*Go*

The Lambda and CodeBuild function code is written in Golang.  We prefer that similar contributed code also be written in Golang.  Please ensure your Golang code is formatted by [gofmt](https://golang.org/cmd/gofmt/) and linted by [golint](https://godoc.org/golang.org/x/lint).

## Pull Requests

If you've gotten as far as reading this section, then thank you for your suggestions.

## Paperwork for Pull Requests

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

## General Guidelines

Ensure your pull request (PR) adheres to the following guidelines:

* Try to make the name concise and descriptive.
* Give a good description of the change being made. Since this is very subjective, see the [Updating Your Pull Request (PR)](#updating-your-pull-request-pr) section below for further details.
* Every pull request should be associated with one or more issues. If no issue exists yet, please create your own.
* Make sure that all applicable issues are mentioned somewhere in the PR description. This can be done by typing # to bring up a list of issues.

### Updating Your Pull Request (PR)

A lot of times, making a PR adhere to the standards above can be difficult. If the maintainers notice anything that we'd like changed, we'll ask you to edit your PR before we merge it. This applies to both the content documented in the PR and the changed contained within the branch being merged. There's no need to open a new PR. Just edit the existing one.

[email]: mailto:opensource@optum.com
