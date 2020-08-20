# Contribution Guidelines

Please note that this project is released with a [Contributor Code of Conduct](CODE_OF_CONDUCT.md). By participating in this project you agree to abide by its terms. Please also review our [Contributor License Agreement ("CLA")](INDIVIDUAL_CONTRIBUTOR_LICENSE.md) prior to submitting changes to the project.  You will need to attest to this agreement following the instructions in the [Paperwork for Pull Requests](#paperwork-for-pull-requests) section below.

---
## How to Contribute

Now that we have the disclaimer out of the way, let's get into how you can be a part of our project. There are many different ways to contribute.

### Issues

We track our work using Issues in GitHub. Feel free to open up your own issue to point out areas for improvement or to suggest your own new experiment. If you are comfortable with signing the waiver linked above and contributing code or documentation, grab your own issue and start working.

### Coding Standards

We have some general guidelines towards contributing to this project.

#### Languages

*Go*

The Lambda and CodeBuild function code is written in Golang.  We prefer that similar contributed code also be written in Golang.  Please ensure your Golang code is formatted by [gofmt](https://golang.org/cmd/gofmt/) and linted by [golint](https://godoc.org/golang.org/x/lint).

### Pull Requests

If you've gotten as far as reading this section, then thank you for your suggestions.

### Paperwork for Pull Requests

* Please read this guide and make sure you agree with our [Contributor License Agreement ("CLA")](INDIVIDUAL_CONTRIBUTOR_LICENSE.md).
* Make sure git knows your name and email address:
   ```
   $ git config user.name "J. Random User"
   $ git config user.email "j.random.user@example.com"
   ```
>The name and email address must be valid as we cannot accept anonymous contributions.
* Write good commit messages.
> Concise commit messages that describe your changes help us better understand your contributions.
* The first time you open a pull request in this repository, you will see a comment on your PR with a link that will allow you to sign our Contributor License Agreement (CLA) if necessary.
> The link will take you to a page that allows you to view our CLA.  You will need to click the `Sign in with GitHub to agree button` and authorize the cla-assistant application to access the email addresses associated with your GitHub account.  Agreeing to the CLA is also considered to be an attestation that you either wrote or have the rights to contribute the code.  All committers to the PR branch will be required to sign the CLA, but you will only need to sign once.  This CLA applies to all repositories in the Optum org.

### General Guidelines

Ensure your pull request (PR) adheres to the following guidelines:

* Try to make the name concise and descriptive.
* Give a good description of the change being made. Since this is very subjective, see the [Updating Your Pull Request (PR)](#updating-your-pull-request-pr) section below for further details.
* Every pull request should be associated with one or more issues. If no issue exists yet, please create your own.
* Make sure that all applicable issues are mentioned somewhere in the PR description. This can be done by typing # to bring up a list of issues.

#### Updating Your Pull Request (PR)

A lot of times, making a PR adhere to the standards above can be difficult. If the maintainers notice anything that we'd like changed, we'll ask you to edit your PR before we merge it. This applies to both the content documented in the PR and the changed contained within the branch being merged. There's no need to open a new PR. Just edit the existing one.

[email]: mailto:opensource@optum.com
