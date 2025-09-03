# Contributing to Score

Thanks for your interest in contributing to Score and helping to improve the project 🎵 

Before you get started, please note that by contributing to this project, you confirm that you are the author of your work, have the necessary rights to contribute it, and that your contribution may be provided under the terms of the [Apache License, Version 2.0](LICENSE).

### Where to Begin!

If you have questions, ideas, or requests about Score, feel free to [open an issue in one of the Score repositories](https://github.com/score-spec) or join the conversation in our Slack community — drop your questions in the `#score` channel of the [CNCF Slack](https://slack.cncf.io/).

We welcome contributions of all kinds:

- Bug fixes and feature development
- Documentation improvements
- Bug and feature reports

### Steps to Contribute
Fixes and improvements can be directly addressed by sending a Pull Request on GitHub. Pull requests will be reviewed by one or more maintainers and merged when acceptable.

We ask that before contributing, please make the effort to coordinate with the maintainers of the project before submitting large or high impact PRs. This will prevent you from doing extra work that may or may not be merged.
### **Sign your work with Developer Certificate of Origin**

To contribute to this project, you must agree to the Developer Certificate of Origin (DCO) for each commit you make. The DCO is a simple statement that you, as a contributor, have the legal right to make the contribution.

See the [DCO](https://developercertificate.org/) file for the full text of what you must agree to.

To successfully sign off your contribution you just add a line to every git commit message:

```git
Signed-off-by: Joe Smith <joe.smith@email.com>
```

Use your real name (sorry, no pseudonyms or anonymous contributions.)

If you set your `user.name` and `user.email` git configs, you can sign your commit automatically with `git commit -s`. You can also use git [aliases](https://git-scm.com/book/tr/v2/Git-Basics-Git-Aliases) like `git config --global alias.ci 'commit -s'`. Now you can commit with git ci and the commit will be signed.

<br />
### **Submitting a Pull Request**

To submit any kinds of improvements, please consider the following:

- Submit an [issue](https://github.com/score-spec/score-compose) describing your proposed change.
- Want to get started? Pick an open issue from our [Good First and Help Wanted Issues](https://clotributor.dev/search?foundation=cncf&project=score).
- Fork this repository, develop and test your changes.
- Create a `feature` branch and submit a pull request against the `main` branch.

### How to Test Code

- Please run `make test` unit tests.
- Your branch can be merged after:
  - CI checks pass
  - Your PR is reviewed and approved by a maintainer (see [MAINTAINERS.md](MAINTAINERS.md))

- If you are new to Score, consider reading our [Documentation](https://github.com/score-dev/docs) and [Examples](https://docs.score.dev/examples/)

## Pull Request Checklist :

- Rebase to the current master branch before submitting your pull request.
- Commits should be as small as possible. Each commit should follow the checklist below:
  - For code changes, add tests relevant to the fixed bug or new feature
  - Pass the compile and tests in CI
  - Commit header (first line) should convey what changed
  - Commit body should include details such as why the changes are required and how the proposed changes (we recommened sharing output)
  - DCO Signed
- If your PR is not getting reviewed or you need a specific person to review it, please reach out to the Score contributors at the [Score slack channel](https://cloud-native.slack.com/archives/C07DN0D1UCW)

### Ensuring all source files contain a license header

A [LICENSE](LICENSE), and [NOTICE](NOTICE) file exists in the root directory, and each source code file should contain
an appropriate Apache 2 header.

To check and update all files, run:

```
$ go install github.com/google/addlicense@latest
$ addlicense -l apache -v -ignore '**/*.yaml' -c Humanitec ./cmd ./internal/
```

## Feature requests
If you have ideas for improving Score, please open an [issue](https://github.com/score-dev/score/issues) and share the use case.

### **Where can I go for help?**

If you need any help, Please tag us on issue or reach out to [us](https://github.com/score-spec/spec?tab=readme-ov-file#-get-in-touch).

### **What does the Code of Conduct mean for me?**

Our [Code of Conduct](CODE_OF_CONDUCT.md) means that you are responsible for treating everyone on the project with respect and courtesy, regardless of their identity. If you are the victim of any inappropriate behavior or comments as described in our Code of Conduct, we are here for you and will do the best to ensure that the abuser is reprimanded appropriately, per our code.
