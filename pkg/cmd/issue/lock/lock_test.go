package lock

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/cli/v2/internal/ghrepo"
	"github.com/cli/cli/v2/internal/prompter"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/httpmock"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/cli/cli/v2/test"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
)

func Test_NewCmdLock(t *testing.T) {
	// TODO parent name stuff?
	cases := []struct {
		name    string
		args    string
		want    LockOptions
		wantErr string
		tty     bool
	}{
		{
			name: "sets reason",
			args: "--reason off_topic 451",
			want: LockOptions{
				Reason:      "off_topic",
				SelectorArg: "451",
			},
		},
		{
			name:    "no args",
			wantErr: "accepts 1 arg(s), received 0",
		},
		{
			name: "no flags",
			args: "451",
			want: LockOptions{
				SelectorArg: "451",
			},
		},
		{
			name:    "bad reason",
			args:    "--reason bad 451",
			wantErr: "invalid reason bad",
		},
		{
			name:    "bad reason tty",
			args:    "--reason bad 451",
			tty:     true,
			wantErr: "X Invalid reason: bad\n",
		},
		{
			name: "interactive",
			args: "451",
			tty:  true,
			want: LockOptions{
				SelectorArg: "451",
				Interactive: true,
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			ios.SetStdoutTTY(tt.tty)
			ios.SetStdinTTY(tt.tty)
			ios.SetStderrTTY(tt.tty)
			f := &cmdutil.Factory{
				IOStreams: ios,
			}
			var opts *LockOptions
			cmd := NewCmdLock(f, "issue", func(_ string, o *LockOptions) error {
				opts = o
				return nil
			})
			cmd.PersistentFlags().StringP("repo", "R", "", "")

			argv, err := shlex.Split(tt.args)
			assert.NoError(t, err)

			cmd.SetArgs(argv)
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			_, err = cmd.ExecuteC()
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.want.Reason, opts.Reason)
			assert.Equal(t, tt.want.SelectorArg, opts.SelectorArg)
			assert.Equal(t, tt.want.Interactive, opts.Interactive)
		})
	}
}

func Test_NewCmdUnlock(t *testing.T) {
	// TODO parent name stuff?
	cases := []struct {
		name    string
		args    string
		want    LockOptions
		wantErr string
		tty     bool
	}{
		{
			name:    "no args",
			wantErr: "accepts 1 arg(s), received 0",
		},
		{
			name: "no flags",
			args: "451",
			want: LockOptions{
				SelectorArg: "451",
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			ios.SetStdoutTTY(tt.tty)
			ios.SetStdinTTY(tt.tty)
			ios.SetStderrTTY(tt.tty)
			f := &cmdutil.Factory{
				IOStreams: ios,
			}
			var opts *LockOptions
			cmd := NewCmdUnlock(f, "issue", func(_ string, o *LockOptions) error {
				opts = o
				return nil
			})
			cmd.PersistentFlags().StringP("repo", "R", "", "")

			argv, err := shlex.Split(tt.args)
			assert.NoError(t, err)

			cmd.SetArgs(argv)
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			_, err = cmd.ExecuteC()
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.want.SelectorArg, opts.SelectorArg)
		})
	}
}

func Test_runLock(t *testing.T) {
	cases := []struct {
		name        string
		opts        LockOptions
		promptStubs func(*testing.T, *prompter.PrompterMock)
		httpStubs   func(*testing.T, *httpmock.Registry)
		wantOut     string
		wantErrOut  string
		wantErr     string
		tty         bool
		state       string
	}{
		{
			name:  "lock an issue nontty",
			state: Lock,
			opts: LockOptions{
				SelectorArg: "451",
				ParentCmd:   "issue",
			},
			httpStubs: func(t *testing.T, reg *httpmock.Registry) {
				reg.Register(
					httpmock.GraphQL(`query IssueByNumber\b`),
					httpmock.StringResponse(`
						{ "data": { "repository": { "hasIssuesEnabled": true, "issue": {
							"number": 451,
							"url": "https://github.com/OWNER/REPO/issues/451",
							"__typename": "Issue" }}}}`))
				reg.Register(
					httpmock.GraphQL(`mutation LockLockable\b`),
					httpmock.StringResponse(`
						{ "data": {
						    "lockLockable": {
						      "lockedRecord": {
						        "locked": true }}}}`))
			},
		},
		{
			name: "lock an issue tty",
			tty:  true,
			opts: LockOptions{
				Interactive: true,
				SelectorArg: "451",
				ParentCmd:   "issue",
			},
			state: Lock,
			httpStubs: func(t *testing.T, reg *httpmock.Registry) {
				reg.Register(
					httpmock.GraphQL(`query IssueByNumber\b`),
					httpmock.StringResponse(`
						{ "data": { "repository": { "hasIssuesEnabled": true, "issue": {
							"number": 451,
							"title": "traverse the library",
							"url": "https://github.com/OWNER/REPO/issues/451",
							"__typename": "Issue" }}}}`))
				reg.Register(
					httpmock.GraphQL(`mutation LockLockable\b`),
					httpmock.StringResponse(`
						{ "data": {
						    "lockLockable": {
						      "lockedRecord": {
						        "locked": true }}}}`))
			},
			promptStubs: func(t *testing.T, pm *prompter.PrompterMock) {
				pm.SelectFunc = func(p, d string, opts []string) (int, error) {
					if p == "Lock reason?" {
						return prompter.IndexFor(opts, "Too heated")
					}

					return -1, prompter.NoSuchPromptErr(p)
				}
			},
			wantOut: "✓ Locked as TOO_HEATED: Issue #451 (traverse the library)\n",
		},
		{
			name:  "lock an issue with explicit reason",
			tty:   true,
			state: Lock,
			opts: LockOptions{
				SelectorArg: "451",
				ParentCmd:   "issue",
				Reason:      "off_topic",
			},
			httpStubs: func(t *testing.T, reg *httpmock.Registry) {
				reg.Register(
					httpmock.GraphQL(`query IssueByNumber\b`),
					httpmock.StringResponse(`
						{ "data": { "repository": { "hasIssuesEnabled": true, "issue": {
							"number": 451,
							"title": "traverse the library",
							"url": "https://github.com/OWNER/REPO/issues/451",
							"__typename": "Issue" }}}}`))
				reg.Register(
					httpmock.GraphQL(`mutation LockLockable\b`),
					httpmock.StringResponse(`
						{ "data": {
						    "lockLockable": {
						      "lockedRecord": {
						        "locked": true }}}}`))
			},
			wantOut: "✓ Locked as OFF_TOPIC: Issue #451 (traverse the library)\n",
		},
		{
			name:  "unlock issue tty",
			tty:   true,
			state: Unlock,
			opts: LockOptions{
				SelectorArg: "451",
				ParentCmd:   "issue",
			},
			httpStubs: func(t *testing.T, reg *httpmock.Registry) {
				reg.Register(
					httpmock.GraphQL(`query IssueByNumber\b`),
					httpmock.StringResponse(`
						{ "data": { "repository": { "hasIssuesEnabled": true, "issue": {
							"number": 451,
							"locked": true,
							"title": "traverse the library",
							"url": "https://github.com/OWNER/REPO/issues/451",
							"__typename": "Issue" }}}}`))
				reg.Register(
					httpmock.GraphQL(`mutation UnlockLockable\b`),
					httpmock.StringResponse(`
						{ "data": {
						    "unlockLockable": {
						      "unlockedRecord": {
						        "locked": false }}}}`))
			},
			wantOut: "✓ Unlocked: Issue #451 (traverse the library)\n",
		},
		{
			name:  "unlock issue nontty",
			state: Lock,
			// TODO
		},
		{
			name:  "lock issue with explicit nontty",
			state: Lock,
			// TODO
		},
		{
			name:  "relock issue tty",
			state: Lock,
			// TODO
		},
		{
			name:  "relock issue nontty",
			state: Lock,
			// TODO
		},

		{
			name: "lock pr nontty",
			// TODO
		},
		{
			name: "lock pr tty",
			// TODO
		},
		{
			name: "lock pr with explicit reason",
			// TODO
		},
		{
			name: "lock pr with explicit nontty",
			// TODO
		},
		{
			name: "unlock pr tty",
			// TODO
		},
		{
			name: "unlock pr nontty",
			// TODO
		},
		{
			name: "relock pr tty",
			// TODO
		},
		{
			name: "relock pr nontty",
			// TODO
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			reg := &httpmock.Registry{}
			defer reg.Verify(t)
			if tt.httpStubs != nil {
				tt.httpStubs(t, reg)
			}

			pm := &prompter.PrompterMock{}
			if tt.promptStubs != nil {
				tt.promptStubs(t, pm)
			}

			ios, _, stdout, stderr := iostreams.Test()
			ios.SetStdoutTTY(tt.tty)
			ios.SetStdinTTY(tt.tty)
			ios.SetStderrTTY(tt.tty)

			tt.opts.Prompter = pm
			tt.opts.IO = ios
			tt.opts.BaseRepo = func() (ghrepo.Interface, error) {
				return ghrepo.FromFullName("OWNER/REPO")
			}
			tt.opts.HttpClient = func() (*http.Client, error) {
				return &http.Client{Transport: reg}, nil
			}

			err := lockRun(tt.state, &tt.opts)
			output := &test.CmdOut{
				OutBuf: stdout,
				ErrBuf: stderr,
			}
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantOut, output.String())
				assert.Equal(t, tt.wantErrOut, output.Stderr())
			}
		})
	}
}

func TestReasons(t *testing.T) {
	assert.Equal(t, len(reasons), len(reasonsApi))

	for _, reason := range reasons {
		assert.Equal(t, strings.ToUpper(reason), string(*reasonsMap[reason]))
	}
}
