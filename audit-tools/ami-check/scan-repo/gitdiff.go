package main

// All code in this file was lifted and tweaked to be in one file from the gitea project

// The notice at the start of their files is:
// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"
	"unicode/utf8"
	"unsafe"

	"github.com/djherbis/buffer"
	"github.com/djherbis/nio/v3"
	"github.com/go-enry/go-enry/v2"
	"github.com/go-enry/go-enry/v2/data"
	"github.com/gogs/chardet"
	stdcharset "golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

type CmdArg string

type TimeStamp int64

type UserType int

type VisibleType int

// User represents the object of individual and member of organization.
type User struct {
	ID        int64  `xorm:"pk autoincr"`
	LowerName string `xorm:"UNIQUE NOT NULL"`
	Name      string `xorm:"UNIQUE NOT NULL"`
	FullName  string
	// Email is the primary email address (to be used for communication)
	Email                        string `xorm:"NOT NULL"`
	KeepEmailPrivate             bool
	EmailNotificationsPreference string `xorm:"VARCHAR(20) NOT NULL DEFAULT 'enabled'"`
	Passwd                       string `xorm:"NOT NULL"`
	PasswdHashAlgo               string `xorm:"NOT NULL DEFAULT 'argon2'"`

	// MustChangePassword is an attribute that determines if a user
	// is to change their password after registration.
	MustChangePassword bool `xorm:"NOT NULL DEFAULT false"`

	LoginType   int
	LoginSource int64 `xorm:"NOT NULL DEFAULT 0"`
	LoginName   string
	Type        UserType
	Location    string
	Website     string
	Rands       string `xorm:"VARCHAR(32)"`
	Salt        string `xorm:"VARCHAR(32)"`
	Language    string `xorm:"VARCHAR(5)"`
	Description string

	CreatedUnix   TimeStamp `xorm:"INDEX created"`
	UpdatedUnix   TimeStamp `xorm:"INDEX updated"`
	LastLoginUnix TimeStamp `xorm:"INDEX"`

	// Remember visibility choice for convenience, true for private
	LastRepoVisibility bool
	// Maximum repository creation limit, -1 means use global default
	MaxRepoCreation int `xorm:"NOT NULL DEFAULT -1"`

	// IsActive true: primary email is activated, user can access Web UI and Git SSH.
	// false: an inactive user can only log in Web UI for account operations (ex: activate the account by email), no other access.
	IsActive bool `xorm:"INDEX"`
	// the user is a Gitea admin, who can access all repositories and the admin pages.
	IsAdmin bool
	// true: the user is only allowed to see organizations/repositories that they has explicit rights to.
	// (ex: in private Gitea instances user won't be allowed to see even organizations/repositories that are set as public)
	IsRestricted bool `xorm:"NOT NULL DEFAULT false"`

	AllowGitHook            bool
	AllowImportLocal        bool // Allow migrate repository by local path
	AllowCreateOrganization bool `xorm:"DEFAULT true"`

	// true: the user is not allowed to log in Web UI. Git/SSH access could still be allowed (please refer to Git/SSH access related code/documents)
	ProhibitLogin bool `xorm:"NOT NULL DEFAULT false"`

	// Avatar
	Avatar          string `xorm:"VARCHAR(2048) NOT NULL"`
	AvatarEmail     string `xorm:"NOT NULL"`
	UseCustomAvatar bool

	// Counters
	NumFollowers int
	NumFollowing int `xorm:"NOT NULL DEFAULT 0"`
	NumStars     int
	NumRepos     int

	// For organization
	NumTeams                  int
	NumMembers                int
	Visibility                VisibleType `xorm:"NOT NULL DEFAULT 0"`
	RepoAdminChangeTeamAccess bool        `xorm:"NOT NULL DEFAULT false"`

	// Preferences
	DiffViewStyle       string `xorm:"NOT NULL DEFAULT ''"`
	Theme               string `xorm:"NOT NULL DEFAULT ''"`
	KeepActivityPrivate bool   `xorm:"NOT NULL DEFAULT false"`
}

type CommentType int

// Label represents a label of repository for issues.
type Label struct {
	ID              int64 `xorm:"pk autoincr"`
	RepoID          int64 `xorm:"INDEX"`
	OrgID           int64 `xorm:"INDEX"`
	Name            string
	Description     string
	Color           string `xorm:"VARCHAR(7)"`
	NumIssues       int
	NumClosedIssues int
	CreatedUnix     TimeStamp `xorm:"INDEX created"`
	UpdatedUnix     TimeStamp `xorm:"INDEX updated"`

	NumOpenIssues     int    `xorm:"-"`
	NumOpenRepoIssues int64  `xorm:"-"`
	IsChecked         bool   `xorm:"-"`
	QueryString       string `xorm:"-"`
	IsSelected        bool   `xorm:"-"`
	IsExcluded        bool   `xorm:"-"`
}

// Milestone represents a milestone of repository.
type Milestone struct {
	ID              int64       `xorm:"pk autoincr"`
	RepoID          int64       `xorm:"INDEX"`
	Repo            *Repository `xorm:"-"`
	Name            string
	Content         string `xorm:"TEXT"`
	RenderedContent string `xorm:"-"`
	IsClosed        bool
	NumIssues       int
	NumClosedIssues int
	NumOpenIssues   int  `xorm:"-"`
	Completeness    int  // Percentage(1-100).
	IsOverdue       bool `xorm:"-"`

	CreatedUnix    TimeStamp `xorm:"INDEX created"`
	UpdatedUnix    TimeStamp `xorm:"INDEX updated"`
	DeadlineUnix   TimeStamp
	ClosedDateUnix TimeStamp
	DeadlineString string `xorm:"-"`

	TotalTrackedTime int64 `xorm:"-"`
	TimeSinceUpdate  int64 `xorm:"-"`
}

type BoardType uint8

// Project represents a project board
type Project struct {
	ID          int64  `xorm:"pk autoincr"`
	Title       string `xorm:"INDEX NOT NULL"`
	Description string `xorm:"TEXT"`
	RepoID      int64  `xorm:"INDEX"`
	CreatorID   int64  `xorm:"NOT NULL"`
	IsClosed    bool   `xorm:"INDEX"`
	BoardType   BoardType
	Type        uint8

	RenderedContent string `xorm:"-"`

	CreatedUnix    TimeStamp `xorm:"INDEX created"`
	UpdatedUnix    TimeStamp `xorm:"INDEX updated"`
	ClosedDateUnix TimeStamp
}

// PullRequestType defines pull request type
type PullRequestType int

// PullRequestStatus defines pull request status
type PullRequestStatus int

// ProtectedBranch struct
type ProtectedBranch struct {
	ID                            int64  `xorm:"pk autoincr"`
	RepoID                        int64  `xorm:"UNIQUE(s)"`
	BranchName                    string `xorm:"UNIQUE(s)"`
	CanPush                       bool   `xorm:"NOT NULL DEFAULT false"`
	EnableWhitelist               bool
	WhitelistUserIDs              []int64  `xorm:"JSON TEXT"`
	WhitelistTeamIDs              []int64  `xorm:"JSON TEXT"`
	EnableMergeWhitelist          bool     `xorm:"NOT NULL DEFAULT false"`
	WhitelistDeployKeys           bool     `xorm:"NOT NULL DEFAULT false"`
	MergeWhitelistUserIDs         []int64  `xorm:"JSON TEXT"`
	MergeWhitelistTeamIDs         []int64  `xorm:"JSON TEXT"`
	EnableStatusCheck             bool     `xorm:"NOT NULL DEFAULT false"`
	StatusCheckContexts           []string `xorm:"JSON TEXT"`
	EnableApprovalsWhitelist      bool     `xorm:"NOT NULL DEFAULT false"`
	ApprovalsWhitelistUserIDs     []int64  `xorm:"JSON TEXT"`
	ApprovalsWhitelistTeamIDs     []int64  `xorm:"JSON TEXT"`
	RequiredApprovals             int64    `xorm:"NOT NULL DEFAULT 0"`
	BlockOnRejectedReviews        bool     `xorm:"NOT NULL DEFAULT false"`
	BlockOnOfficialReviewRequests bool     `xorm:"NOT NULL DEFAULT false"`
	BlockOnOutdatedBranch         bool     `xorm:"NOT NULL DEFAULT false"`
	DismissStaleApprovals         bool     `xorm:"NOT NULL DEFAULT false"`
	RequireSignedCommits          bool     `xorm:"NOT NULL DEFAULT false"`
	ProtectedFilePatterns         string   `xorm:"TEXT"`
	UnprotectedFilePatterns       string   `xorm:"TEXT"`

	CreatedUnix TimeStamp `xorm:"created"`
	UpdatedUnix TimeStamp `xorm:"updated"`
}

// PullRequestFlow the flow of pull request
type PullRequestFlow int

// PullRequest represents relation between pull request and repositories.
type PullRequest struct {
	ID              int64 `xorm:"pk autoincr"`
	Type            PullRequestType
	Status          PullRequestStatus
	ConflictedFiles []string `xorm:"TEXT JSON"`
	CommitsAhead    int
	CommitsBehind   int

	ChangedProtectedFiles []string `xorm:"TEXT JSON"`

	IssueID int64  `xorm:"INDEX"`
	Issue   *Issue `xorm:"-"`
	Index   int64

	HeadRepoID          int64       `xorm:"INDEX"`
	HeadRepo            *Repository `xorm:"-"`
	BaseRepoID          int64       `xorm:"INDEX"`
	BaseRepo            *Repository `xorm:"-"`
	HeadBranch          string
	HeadCommitID        string `xorm:"-"`
	BaseBranch          string
	ProtectedBranch     *ProtectedBranch `xorm:"-"`
	MergeBase           string           `xorm:"VARCHAR(40)"`
	AllowMaintainerEdit bool             `xorm:"NOT NULL DEFAULT false"`

	HasMerged      bool      `xorm:"INDEX"`
	MergedCommitID string    `xorm:"VARCHAR(40)"`
	MergerID       int64     `xorm:"INDEX"`
	Merger         *User     `xorm:"-"`
	MergedUnix     TimeStamp `xorm:"updated INDEX"`

	isHeadRepoLoaded bool `xorm:"-"`

	Flow PullRequestFlow `xorm:"NOT NULL DEFAULT 0"`
}

// Attachment represent a attachment of issue/comment/release.
type Attachment struct {
	ID            int64  `xorm:"pk autoincr"`
	UUID          string `xorm:"uuid UNIQUE"`
	RepoID        int64  `xorm:"INDEX"`           // this should not be zero
	IssueID       int64  `xorm:"INDEX"`           // maybe zero when creating
	ReleaseID     int64  `xorm:"INDEX"`           // maybe zero when creating
	UploaderID    int64  `xorm:"INDEX DEFAULT 0"` // Notice: will be zero before this column added
	CommentID     int64
	Name          string
	DownloadCount int64     `xorm:"DEFAULT 0"`
	Size          int64     `xorm:"DEFAULT 0"`
	CreatedUnix   TimeStamp `xorm:"created"`
}

// Reaction represents a reactions on issues and comments.
type Reaction struct {
	ID               int64     `xorm:"pk autoincr"`
	Type             string    `xorm:"INDEX UNIQUE(s) NOT NULL"`
	IssueID          int64     `xorm:"INDEX UNIQUE(s) NOT NULL"`
	CommentID        int64     `xorm:"INDEX UNIQUE(s)"`
	UserID           int64     `xorm:"INDEX UNIQUE(s) NOT NULL"`
	OriginalAuthorID int64     `xorm:"INDEX UNIQUE(s) NOT NULL DEFAULT(0)"`
	OriginalAuthor   string    `xorm:"INDEX UNIQUE(s)"`
	User             *User     `xorm:"-"`
	CreatedUnix      TimeStamp `xorm:"INDEX created"`
}

// ReactionList represents list of reactions
type ReactionList []*Reaction

// Type* are valid values for the Type field of ForeignReference
const (
	TypeIssue         = "issue"
	TypePullRequest   = "pull_request"
	TypeComment       = "comment"
	TypeReview        = "review"
	TypeReviewComment = "review_comment"
	TypeRelease       = "release"
)

// ForeignReference represents external references
type ForeignReference struct {
	// RepoID is the first column in all indices. now we only need 2 indices: (repo, local) and (repo, foreign, type)
	RepoID       int64  `xorm:"UNIQUE(repo_foreign_type) INDEX(repo_local)" `
	LocalIndex   int64  `xorm:"INDEX(repo_local)"` // the resource key inside Gitea, it can be IssueIndex, or some model ID.
	ForeignIndex string `xorm:"INDEX UNIQUE(repo_foreign_type)"`
	Type         string `xorm:"VARCHAR(16) INDEX UNIQUE(repo_foreign_type)"`
}

// RoleDescriptor defines comment tag type
type RoleDescriptor int

// Issue represents an issue or pull request of repository.
type Issue struct {
	ID               int64       `xorm:"pk autoincr"`
	RepoID           int64       `xorm:"INDEX UNIQUE(repo_index)"`
	Repo             *Repository `xorm:"-"`
	Index            int64       `xorm:"UNIQUE(repo_index)"` // Index in one repository.
	PosterID         int64       `xorm:"INDEX"`
	Poster           *User       `xorm:"-"`
	OriginalAuthor   string
	OriginalAuthorID int64      `xorm:"index"`
	Title            string     `xorm:"name"`
	Content          string     `xorm:"LONGTEXT"`
	RenderedContent  string     `xorm:"-"`
	Labels           []*Label   `xorm:"-"`
	MilestoneID      int64      `xorm:"INDEX"`
	Milestone        *Milestone `xorm:"-"`
	Project          *Project   `xorm:"-"`
	Priority         int
	AssigneeID       int64        `xorm:"-"`
	Assignee         *User        `xorm:"-"`
	IsClosed         bool         `xorm:"INDEX"`
	IsRead           bool         `xorm:"-"`
	IsPull           bool         `xorm:"INDEX"` // Indicates whether is a pull request or not.
	PullRequest      *PullRequest `xorm:"-"`
	NumComments      int
	Ref              string

	DeadlineUnix TimeStamp `xorm:"INDEX"`

	CreatedUnix TimeStamp `xorm:"INDEX created"`
	UpdatedUnix TimeStamp `xorm:"INDEX updated"`
	ClosedUnix  TimeStamp `xorm:"INDEX"`

	Attachments      []*Attachment     `xorm:"-"`
	Comments         []*Comment        `xorm:"-"`
	Reactions        ReactionList      `xorm:"-"`
	TotalTrackedTime int64             `xorm:"-"`
	Assignees        []*User           `xorm:"-"`
	ForeignReference *ForeignReference `xorm:"-"`

	// IsLocked limits commenting abilities to users on an issue
	// with write access
	IsLocked bool `xorm:"NOT NULL DEFAULT false"`

	// For view issue page.
	ShowRole RoleDescriptor `xorm:"-"`
}

// TrackedTime represents a time that was spent for a specific issue.
type TrackedTime struct {
	ID          int64     `xorm:"pk autoincr"`
	IssueID     int64     `xorm:"INDEX"`
	Issue       *Issue    `xorm:"-"`
	UserID      int64     `xorm:"INDEX"`
	User        *User     `xorm:"-"`
	Created     time.Time `xorm:"-"`
	CreatedUnix int64     `xorm:"created"`
	Time        int64     `xorm:"NOT NULL"`
	Deleted     bool      `xorm:"NOT NULL DEFAULT false"`
}

// OwnerTeamName return the owner team name
const OwnerTeamName = "Owners"

// AccessMode specifies the users access mode
type AccessMode int

const (
	// AccessModeNone no access
	AccessModeNone AccessMode = iota // 0
	// AccessModeRead read access
	AccessModeRead // 1
	// AccessModeWrite write access
	AccessModeWrite // 2
	// AccessModeAdmin admin access
	AccessModeAdmin // 3
	// AccessModeOwner owner access
	AccessModeOwner // 4
)

// TeamUnit describes all units of a repository
type TeamUnit struct {
	ID         int64 `xorm:"pk autoincr"`
	OrgID      int64 `xorm:"INDEX"`
	TeamID     int64 `xorm:"UNIQUE(s)"`
	Type       int   `xorm:"UNIQUE(s)"`
	AccessMode AccessMode
}

// Team represents a organization team.
type Team struct {
	ID                      int64 `xorm:"pk autoincr"`
	OrgID                   int64 `xorm:"INDEX"`
	LowerName               string
	Name                    string
	Description             string
	AccessMode              AccessMode    `xorm:"'authorize'"`
	Repos                   []*Repository `xorm:"-"`
	Members                 []*User       `xorm:"-"`
	NumRepos                int
	NumMembers              int
	Units                   []*TeamUnit `xorm:"-"`
	IncludesAllRepositories bool        `xorm:"NOT NULL DEFAULT false"`
	CanCreateOrgRepo        bool        `xorm:"NOT NULL DEFAULT false"`
}

type ReviewType int

// CodeComments represents comments on code by using this structure: FILENAME -> LINE (+ == proposed; - == previous) -> COMMENTS
type CodeComments map[string]map[int64][]*Comment

// Review represents collection of code comments giving feedback for a PR
type Review struct {
	ID               int64 `xorm:"pk autoincr"`
	Type             ReviewType
	Reviewer         *User `xorm:"-"`
	ReviewerID       int64 `xorm:"index"`
	ReviewerTeamID   int64 `xorm:"NOT NULL DEFAULT 0"`
	ReviewerTeam     *Team `xorm:"-"`
	OriginalAuthor   string
	OriginalAuthorID int64
	Issue            *Issue `xorm:"-"`
	IssueID          int64  `xorm:"index"`
	Content          string `xorm:"TEXT"`
	// Official is a review made by an assigned approver (counts towards approval)
	Official  bool   `xorm:"NOT NULL DEFAULT false"`
	CommitID  string `xorm:"VARCHAR(40)"`
	Stale     bool   `xorm:"NOT NULL DEFAULT false"`
	Dismissed bool   `xorm:"NOT NULL DEFAULT false"`

	CreatedUnix TimeStamp `xorm:"INDEX created"`
	UpdatedUnix TimeStamp `xorm:"INDEX updated"`

	// CodeComments are the initial code comments of the review
	CodeComments CodeComments `xorm:"-"`

	Comments []*Comment `xorm:"-"`
}

// CommitStatusState holds the state of a CommitStatus
// It can be "pending", "success", "error", "failure", and "warning"
type CommitStatusState string

const (
	// CommitStatusPending is for when the CommitStatus is Pending
	CommitStatusPending CommitStatusState = "pending"
	// CommitStatusSuccess is for when the CommitStatus is Success
	CommitStatusSuccess CommitStatusState = "success"
	// CommitStatusError is for when the CommitStatus is Error
	CommitStatusError CommitStatusState = "error"
	// CommitStatusFailure is for when the CommitStatus is Failure
	CommitStatusFailure CommitStatusState = "failure"
	// CommitStatusWarning is for when the CommitStatus is Warning
	CommitStatusWarning CommitStatusState = "warning"
)

// CommitStatus holds a single Status of a single Commit
type CommitStatus struct {
	ID          int64             `xorm:"pk autoincr"`
	Index       int64             `xorm:"INDEX UNIQUE(repo_sha_index)"`
	RepoID      int64             `xorm:"INDEX UNIQUE(repo_sha_index)"`
	Repo        *Repository       `xorm:"-"`
	State       CommitStatusState `xorm:"VARCHAR(7) NOT NULL"`
	SHA         string            `xorm:"VARCHAR(64) NOT NULL INDEX UNIQUE(repo_sha_index)"`
	TargetURL   string            `xorm:"TEXT"`
	Description string            `xorm:"TEXT"`
	ContextHash string            `xorm:"char(40) index"`
	Context     string            `xorm:"TEXT"`
	Creator     *User             `xorm:"-"`
	CreatorID   int64

	CreatedUnix TimeStamp `xorm:"INDEX created"`
	UpdatedUnix TimeStamp `xorm:"INDEX updated"`
}

// EmailAddress is the list of all email addresses of a user. It also contains the
// primary email address which is saved in user table.
type EmailAddress struct {
	ID          int64  `xorm:"pk autoincr"`
	UID         int64  `xorm:"INDEX NOT NULL"`
	Email       string `xorm:"UNIQUE NOT NULL"`
	LowerEmail  string `xorm:"UNIQUE NOT NULL"`
	IsActivated bool
	IsPrimary   bool `xorm:"DEFAULT(false) NOT NULL"`
}

// GPGKey represents a GPG key.
type GPGKey struct {
	ID                int64     `xorm:"pk autoincr"`
	OwnerID           int64     `xorm:"INDEX NOT NULL"`
	KeyID             string    `xorm:"INDEX CHAR(16) NOT NULL"`
	PrimaryKeyID      string    `xorm:"CHAR(16)"`
	Content           string    `xorm:"MEDIUMTEXT NOT NULL"`
	CreatedUnix       TimeStamp `xorm:"created"`
	ExpiredUnix       TimeStamp
	AddedUnix         TimeStamp
	SubsKey           []*GPGKey `xorm:"-"`
	Emails            []*EmailAddress
	Verified          bool `xorm:"NOT NULL DEFAULT false"`
	CanSign           bool
	CanEncryptComms   bool
	CanEncryptStorage bool
	CanCertify        bool
}

// KeyType specifies the key type
type KeyType int

const (
	// KeyTypeUser specifies the user key
	KeyTypeUser = iota + 1
	// KeyTypeDeploy specifies the deploy key
	KeyTypeDeploy
	// KeyTypePrincipal specifies the authorized principal key
	KeyTypePrincipal
)

// PublicKey represents a user or deploy SSH public key.
type PublicKey struct {
	ID            int64      `xorm:"pk autoincr"`
	OwnerID       int64      `xorm:"INDEX NOT NULL"`
	Name          string     `xorm:"NOT NULL"`
	Fingerprint   string     `xorm:"INDEX NOT NULL"`
	Content       string     `xorm:"MEDIUMTEXT NOT NULL"`
	Mode          AccessMode `xorm:"NOT NULL DEFAULT 2"`
	Type          KeyType    `xorm:"NOT NULL DEFAULT 1"`
	LoginSourceID int64      `xorm:"NOT NULL DEFAULT 0"`

	CreatedUnix       TimeStamp `xorm:"created"`
	UpdatedUnix       TimeStamp `xorm:"updated"`
	HasRecentActivity bool      `xorm:"-"`
	HasUsed           bool      `xorm:"-"`
	Verified          bool      `xorm:"NOT NULL DEFAULT false"`
}

// CommitVerification represents a commit validation of signature
type CommitVerification struct {
	Verified       bool
	Warning        bool
	Reason         string
	SigningUser    *User
	CommittingUser *User
	SigningEmail   string
	SigningKey     *GPGKey
	SigningSSHKey  *PublicKey
	TrustStatus    string
}

// UserCommit represents a commit with validation of user.
type UserCommit struct { //revive:disable-line:exported
	User *User
	*Commit
}

// SignCommit represents a commit with validation of signature.
type SignCommit struct {
	Verification *CommitVerification
	*UserCommit
}

// SignCommitWithStatuses represents a commit with validation of signature and status state.
type SignCommitWithStatuses struct {
	Status   *CommitStatus
	Statuses []*CommitStatus
	*SignCommit
}

// XRefAction represents the kind of effect a cross reference has once is resolved
type XRefAction int64

// Comment represents a comment in commit and issue page.
type Comment struct {
	ID               int64       `xorm:"pk autoincr"`
	Type             CommentType `xorm:"INDEX"`
	PosterID         int64       `xorm:"INDEX"`
	Poster           *User       `xorm:"-"`
	OriginalAuthor   string
	OriginalAuthorID int64
	IssueID          int64  `xorm:"INDEX"`
	Issue            *Issue `xorm:"-"`
	LabelID          int64
	Label            *Label   `xorm:"-"`
	AddedLabels      []*Label `xorm:"-"`
	RemovedLabels    []*Label `xorm:"-"`
	OldProjectID     int64
	ProjectID        int64
	OldProject       *Project `xorm:"-"`
	Project          *Project `xorm:"-"`
	OldMilestoneID   int64
	MilestoneID      int64
	OldMilestone     *Milestone `xorm:"-"`
	Milestone        *Milestone `xorm:"-"`
	TimeID           int64
	Time             *TrackedTime `xorm:"-"`
	AssigneeID       int64
	RemovedAssignee  bool
	Assignee         *User `xorm:"-"`
	AssigneeTeamID   int64 `xorm:"NOT NULL DEFAULT 0"`
	AssigneeTeam     *Team `xorm:"-"`
	ResolveDoerID    int64
	ResolveDoer      *User `xorm:"-"`
	OldTitle         string
	NewTitle         string
	OldRef           string
	NewRef           string
	DependentIssueID int64
	DependentIssue   *Issue `xorm:"-"`

	CommitID        int64
	Line            int64 // - previous line / + proposed line
	TreePath        string
	Content         string `xorm:"LONGTEXT"`
	RenderedContent string `xorm:"-"`

	// Path represents the 4 lines of code cemented by this comment
	Patch       string `xorm:"-"`
	PatchQuoted string `xorm:"LONGTEXT patch"`

	CreatedUnix TimeStamp `xorm:"INDEX created"`
	UpdatedUnix TimeStamp `xorm:"INDEX updated"`

	// Reference issue in commit message
	CommitSHA string `xorm:"VARCHAR(40)"`

	Attachments []*Attachment `xorm:"-"`
	Reactions   ReactionList  `xorm:"-"`

	// For view issue page.
	ShowRole RoleDescriptor `xorm:"-"`

	Review      *Review `xorm:"-"`
	ReviewID    int64   `xorm:"index"`
	Invalidated bool

	// Reference an issue or pull from another comment, issue or PR
	// All information is about the origin of the reference
	RefRepoID    int64      `xorm:"index"` // Repo where the referencing
	RefIssueID   int64      `xorm:"index"`
	RefCommentID int64      `xorm:"index"`    // 0 if origin is Issue title or content (or PR's)
	RefAction    XRefAction `xorm:"SMALLINT"` // What happens if RefIssueID resolves
	RefIsPull    bool

	RefRepo    *Repository `xorm:"-"`
	RefIssue   *Issue      `xorm:"-"`
	RefComment *Comment    `xorm:"-"`

	Commits     []*SignCommitWithStatuses `xorm:"-"`
	OldCommit   string                    `xorm:"-"`
	NewCommit   string                    `xorm:"-"`
	CommitsNum  int64                     `xorm:"-"`
	IsForcePush bool                      `xorm:"-"`
}

// DiffOptions represents the options for a DiffRange
type DiffOptions struct {
	BeforeCommitID     string
	AfterCommitID      string
	SkipTo             string
	MaxLines           int
	MaxLineCharacters  int
	MaxFiles           int
	WhitespaceBehavior CmdArg
	DirectComparison   bool
}

// DiffLineType represents the type of DiffLine.
type DiffLineType uint8

// DiffLineType possible values.
const (
	DiffLinePlain DiffLineType = iota + 1
	DiffLineAdd
	DiffLineDel
	DiffLineSection
)

// DiffLine represents a line difference in a DiffSection.
type DiffLine struct {
	LeftIdx     int
	RightIdx    int
	Match       int
	Type        DiffLineType
	Content     string
	Comments    []*Comment
	SectionInfo *DiffLineSectionInfo
}

// DiffLineSectionInfo represents diff line section meta data
type DiffLineSectionInfo struct {
	Path          string
	LastLeftIdx   int
	LastRightIdx  int
	LeftIdx       int
	RightIdx      int
	LeftHunkSize  int
	RightHunkSize int
}

// DiffSection represents a section of a DiffFile.
type DiffSection struct {
	file     *DiffFile
	FileName string
	Name     string
	Lines    []*DiffLine
}

// DiffFileType represents the type of DiffFile.
type DiffFileType uint8

// DiffFileType possible values.
const (
	DiffFileAdd DiffFileType = iota + 1
	DiffFileChange
	DiffFileDel
	DiffFileRename
	DiffFileCopy
)

// DiffFile represents a file diff.
type DiffFile struct {
	Name                      string
	NameHash                  string
	OldName                   string
	Index                     int
	Addition, Deletion        int
	Type                      DiffFileType
	IsCreated                 bool
	IsDeleted                 bool
	IsBin                     bool
	IsLFSFile                 bool
	IsRenamed                 bool
	IsAmbiguous               bool
	IsSubmodule               bool
	Sections                  []*DiffSection
	IsIncomplete              bool
	IsIncompleteLineTooLong   bool
	IsProtected               bool
	IsGenerated               bool
	IsVendored                bool
	IsViewed                  bool // User specific
	HasChangedSinceLastReview bool // User specific
	Language                  string
}

type Diff struct {
	Start, End                   string
	NumFiles                     int
	TotalAddition, TotalDeletion int
	Files                        []*DiffFile
	IsIncomplete                 bool
	NumViewedFiles               int // user-specific
}

// ObjectCache provides thread-safe cache operations.
type ObjectCache struct {
	lock  sync.RWMutex
	cache map[string]interface{}
}

// GPGSettings represents the default GPG settings for this repository
type GPGSettings struct {
	Sign             bool
	KeyID            string
	Email            string
	Name             string
	PublicKeyContent string
}

// WriteCloserError wraps an io.WriteCloser with an additional CloseWithError function
type WriteCloserError interface {
	io.WriteCloser
	CloseWithError(err error) error
}

// SHA1 a git commit name
type SHA1 [20]byte

// TreeEntry the leaf in the git tree
type TreeEntry struct {
	ID SHA1

	ptree *Tree

	entryMode EntryMode
	name      string

	size     int64
	sized    bool
	fullName string
}

// Name returns the name of the entry
func (te *TreeEntry) Name() string {
	if te.fullName != "" {
		return te.fullName
	}
	return te.name
}

// Entries a list of entry
type Entries []*TreeEntry

// Tree represents a flat directory listing.
type Tree struct {
	ID         SHA1
	ResolvedID SHA1
	repo       *Repository

	// parent tree
	ptree *Tree

	entries       Entries
	entriesParsed bool

	entriesRecursive       Entries
	entriesRecursiveParsed bool
}

// Signature represents the Author or Committer information.
type Signature struct {
	// Name represents a person name. It is an arbitrary string.
	Name string
	// Email is an email, but it cannot be assumed to be well-formed.
	Email string
	// When is the timestamp of the signature.
	When time.Time
}

// CommitGPGSignature represents a git commit signature part.
type CommitGPGSignature struct {
	Signature string
	Payload   string // TODO check if can be reconstruct from the rest of commit information to not have duplicate data
}

// Commit represents a git commit.
type Commit struct {
	Branch string // Branch this commit belongs to
	Tree
	ID            SHA1 // The ID of this commit object
	Author        *Signature
	Committer     *Signature
	CommitMessage string
	Signature     *CommitGPGSignature

	Parents        []SHA1 // SHA1 strings
	submoduleCache *ObjectCache
}

// Cache represents a caching interface
type Cache interface {
	// Put puts value into cache with key and expire time.
	Put(key string, val interface{}, timeout int64) error
	// Get gets cached value by given key.
	Get(key string) interface{}
}

// LastCommitCache represents a cache to store last commit
type LastCommitCache struct {
	repoPath    string
	ttl         func() int64
	repo        *Repository
	commitCache map[string]*Commit
	cache       Cache
}

// Repository represents a Git repository.
type Repository struct {
	Path string

	tagCache *ObjectCache

	gpgSettings *GPGSettings

	batchCancel context.CancelFunc
	batchReader *bufio.Reader
	batchWriter WriteCloserError

	checkCancel context.CancelFunc
	checkReader *bufio.Reader
	checkWriter WriteCloserError

	Ctx             context.Context
	LastCommitCache *LastCommitCache
}

// enumerates all the policy repository creating
const (
	RepoCreatingLastUserVisibility = "last"
	RepoCreatingPrivate            = "private"
	RepoCreatingPublic             = "public"
)

// ItemsPerPage maximum items per page in forks, watchers and stars of a repo
const ItemsPerPage = 40

// Repository settings
var (
	setting_Repository = struct {
		DetectedCharsetsOrder                   []string
		DetectedCharsetScore                    map[string]int `ini:"-"`
		AnsiCharset                             string
		ForcePrivate                            bool
		DefaultPrivate                          string
		DefaultPushCreatePrivate                bool
		MaxCreationLimit                        int
		PreferredLicenses                       []string
		DisableHTTPGit                          bool
		AccessControlAllowOrigin                string
		UseCompatSSHURI                         bool
		DefaultCloseIssuesViaCommitsInAnyBranch bool
		EnablePushCreateUser                    bool
		EnablePushCreateOrg                     bool
		DisabledRepoUnits                       []string
		DefaultRepoUnits                        []string
		PrefixArchiveFiles                      bool
		DisableMigrations                       bool
		DisableStars                            bool `ini:"DISABLE_STARS"`
		DefaultBranch                           string
		AllowAdoptionOfUnadoptedRepositories    bool
		AllowDeleteOfUnadoptedRepositories      bool
		DisableDownloadSourceArchives           bool

		// Repository editor settings
		Editor struct {
			LineWrapExtensions   []string
			PreviewableFileModes []string
		} `ini:"-"`

		// Repository upload settings
		Upload struct {
			Enabled      bool
			TempPath     string
			AllowedTypes string
			FileMaxSize  int64
			MaxFiles     int
		} `ini:"-"`

		// Repository local settings
		Local struct {
			LocalCopyPath string
		} `ini:"-"`

		// Pull request settings
		PullRequest struct {
			WorkInProgressPrefixes                   []string
			CloseKeywords                            []string
			ReopenKeywords                           []string
			DefaultMergeStyle                        string
			DefaultMergeMessageCommitsLimit          int
			DefaultMergeMessageSize                  int
			DefaultMergeMessageAllAuthors            bool
			DefaultMergeMessageMaxApprovers          int
			DefaultMergeMessageOfficialApproversOnly bool
			PopulateSquashCommentWithCommitMessages  bool
			AddCoCommitterTrailers                   bool
		} `ini:"repository.pull-request"`

		// Issue Setting
		Issue struct {
			LockReasons []string
		} `ini:"repository.issue"`

		Release struct {
			AllowedTypes     string
			DefaultPagingNum int
		} `ini:"repository.release"`

		Signing struct {
			SigningKey        string
			SigningName       string
			SigningEmail      string
			InitialCommit     []string
			CRUDActions       []string `ini:"CRUD_ACTIONS"`
			Merges            []string
			Wiki              []string
			DefaultTrustModel string
		} `ini:"repository.signing"`
	}{
		DetectedCharsetsOrder: []string{
			"UTF-8",
			"UTF-16BE",
			"UTF-16LE",
			"UTF-32BE",
			"UTF-32LE",
			"ISO-8859-1",
			"windows-1252",
			"ISO-8859-2",
			"windows-1250",
			"ISO-8859-5",
			"ISO-8859-6",
			"ISO-8859-7",
			"windows-1253",
			"ISO-8859-8-I",
			"windows-1255",
			"ISO-8859-8",
			"windows-1251",
			"windows-1256",
			"KOI8-R",
			"ISO-8859-9",
			"windows-1254",
			"Shift_JIS",
			"GB18030",
			"EUC-JP",
			"EUC-KR",
			"Big5",
			"ISO-2022-JP",
			"ISO-2022-KR",
			"ISO-2022-CN",
			"IBM424_rtl",
			"IBM424_ltr",
			"IBM420_rtl",
			"IBM420_ltr",
		},
		DetectedCharsetScore:                    map[string]int{},
		AnsiCharset:                             "",
		ForcePrivate:                            false,
		DefaultPrivate:                          RepoCreatingLastUserVisibility,
		DefaultPushCreatePrivate:                true,
		MaxCreationLimit:                        -1,
		PreferredLicenses:                       []string{"Apache License 2.0", "MIT License"},
		DisableHTTPGit:                          false,
		AccessControlAllowOrigin:                "",
		UseCompatSSHURI:                         false,
		DefaultCloseIssuesViaCommitsInAnyBranch: false,
		EnablePushCreateUser:                    false,
		EnablePushCreateOrg:                     false,
		DisabledRepoUnits:                       []string{},
		DefaultRepoUnits:                        []string{},
		PrefixArchiveFiles:                      true,
		DisableMigrations:                       false,
		DisableStars:                            false,
		DefaultBranch:                           "main",

		// Repository editor settings
		Editor: struct {
			LineWrapExtensions   []string
			PreviewableFileModes []string
		}{
			LineWrapExtensions:   strings.Split(".txt,.md,.markdown,.mdown,.mkd,", ","),
			PreviewableFileModes: []string{"markdown"},
		},

		// Repository upload settings
		Upload: struct {
			Enabled      bool
			TempPath     string
			AllowedTypes string
			FileMaxSize  int64
			MaxFiles     int
		}{
			Enabled:      true,
			TempPath:     "data/tmp/uploads",
			AllowedTypes: "",
			FileMaxSize:  3,
			MaxFiles:     5,
		},

		// Repository local settings
		Local: struct {
			LocalCopyPath string
		}{
			LocalCopyPath: "tmp/local-repo",
		},

		// Pull request settings
		PullRequest: struct {
			WorkInProgressPrefixes                   []string
			CloseKeywords                            []string
			ReopenKeywords                           []string
			DefaultMergeStyle                        string
			DefaultMergeMessageCommitsLimit          int
			DefaultMergeMessageSize                  int
			DefaultMergeMessageAllAuthors            bool
			DefaultMergeMessageMaxApprovers          int
			DefaultMergeMessageOfficialApproversOnly bool
			PopulateSquashCommentWithCommitMessages  bool
			AddCoCommitterTrailers                   bool
		}{
			WorkInProgressPrefixes: []string{"WIP:", "[WIP]"},
			// Same as GitHub. See
			// https://help.github.com/articles/closing-issues-via-commit-messages
			CloseKeywords:                            strings.Split("close,closes,closed,fix,fixes,fixed,resolve,resolves,resolved", ","),
			ReopenKeywords:                           strings.Split("reopen,reopens,reopened", ","),
			DefaultMergeStyle:                        "merge",
			DefaultMergeMessageCommitsLimit:          50,
			DefaultMergeMessageSize:                  5 * 1024,
			DefaultMergeMessageAllAuthors:            false,
			DefaultMergeMessageMaxApprovers:          10,
			DefaultMergeMessageOfficialApproversOnly: true,
			PopulateSquashCommentWithCommitMessages:  false,
			AddCoCommitterTrailers:                   true,
		},

		// Issue settings
		Issue: struct {
			LockReasons []string
		}{
			LockReasons: strings.Split("Too heated,Off-topic,Spam,Resolved", ","),
		},

		Release: struct {
			AllowedTypes     string
			DefaultPagingNum int
		}{
			AllowedTypes:     "",
			DefaultPagingNum: 10,
		},

		// Signing settings
		Signing: struct {
			SigningKey        string
			SigningName       string
			SigningEmail      string
			InitialCommit     []string
			CRUDActions       []string `ini:"CRUD_ACTIONS"`
			Merges            []string
			Wiki              []string
			DefaultTrustModel string
		}{
			SigningKey:        "default",
			SigningName:       "",
			SigningEmail:      "",
			InitialCommit:     []string{"always"},
			CRUDActions:       []string{"pubkey", "twofa", "parentsigned"},
			Merges:            []string{"pubkey", "twofa", "basesigned", "commitssigned"},
			Wiki:              []string{"never"},
			DefaultTrustModel: "collaborator",
		},
	}
	RepoRootPath string
	ScriptType   = "bash"

//	RepoArchive = struct {
//		Storage
//	}{}
)

/*
// GitServiceType represents a git service
type GitServiceType int

// enumerate all GitServiceType
const (
	NotMigrated      GitServiceType = iota // 0 not migrated from external sites
	PlainGitService                        // 1 plain git service
	GithubService                          // 2 github.com
	GiteaService                           // 3 gitea service
	GitlabService                          // 4 gitlab service
	GogsService                            // 5 gogs service
	OneDevService                          // 6 onedev service
	GitBucketService                       // 7 gitbucket service
	CodebaseService                        // 8 codebase service
)

// Mirror represents mirror information of a repository.
type Mirror struct {
	ID          int64       `xorm:"pk autoincr"`
	RepoID      int64       `xorm:"INDEX"`
	Repo        *Repository `xorm:"-"`
	Interval    time.Duration
	EnablePrune bool `xorm:"NOT NULL DEFAULT true"`

	UpdatedUnix    TimeStamp `xorm:"INDEX"`
	NextUpdateUnix TimeStamp `xorm:"INDEX"`

	LFS         bool   `xorm:"lfs_enabled NOT NULL DEFAULT false"`
	LFSEndpoint string `xorm:"lfs_endpoint TEXT"`

	Address string `xorm:"-"`
}

// RepositoryStatus defines the status of repository
type RepositoryStatus int

// Conversion is an interface. A type implements Conversion will according
// the custom method to fill into database and retrieve from database.
type Conversion interface {
	FromDB([]byte) error
	ToDB() ([]byte, error)
}

// RepoUnit describes all units of a repository
type RepoUnit struct { //revive:disable-line:exported
	ID          int64
	RepoID      int64      `xorm:"INDEX(s)"`
	Type        int        `xorm:"INDEX(s)"`
	Config      Conversion `xorm:"TEXT"`
	CreatedUnix TimeStamp  `xorm:"INDEX CREATED"`
}

// LanguageStat describes language statistics of a repository
type LanguageStat struct {
	ID          int64 `xorm:"pk autoincr"`
	RepoID      int64 `xorm:"UNIQUE(s) INDEX NOT NULL"`
	CommitID    string
	IsPrimary   bool
	Language    string    `xorm:"VARCHAR(50) UNIQUE(s) INDEX NOT NULL"`
	Percentage  float32   `xorm:"-"`
	Size        int64     `xorm:"NOT NULL DEFAULT 0"`
	Color       string    `xorm:"-"`
	CreatedUnix TimeStamp `xorm:"INDEX CREATED"`
}

// RepoIndexerType specifies the repository indexer type
type RepoIndexerType int //revive:disable-line:exported

const (
	// RepoIndexerTypeCode code indexer
	RepoIndexerTypeCode RepoIndexerType = iota // 0
	// RepoIndexerTypeStats repository stats indexer
	RepoIndexerTypeStats // 1
)

// RepoIndexerStatus status of a repo's entry in the repo indexer
// For now, implicitly refers to default branch
type RepoIndexerStatus struct { //revive:disable-line:exported
	ID          int64           `xorm:"pk autoincr"`
	RepoID      int64           `xorm:"INDEX(s)"`
	CommitSha   string          `xorm:"VARCHAR(40)"`
	IndexerType RepoIndexerType `xorm:"INDEX(s) NOT NULL DEFAULT 0"`
}

// TrustModelType defines the types of trust model for this repository
type TrustModelType int

// kinds of TrustModel
const (
	DefaultTrustModel TrustModelType = iota // default trust model
	CommitterTrustModel
	CollaboratorTrustModel
	CollaboratorCommitterTrustModel
)*/

/*// Repository represents a git repository.
type repo_Repository struct {
	ID                  int64 `xorm:"pk autoincr"`
	OwnerID             int64 `xorm:"UNIQUE(s) index"`
	OwnerName           string
	Owner               *User          `xorm:"-"`
	LowerName           string         `xorm:"UNIQUE(s) INDEX NOT NULL"`
	Name                string         `xorm:"INDEX NOT NULL"`
	Description         string         `xorm:"TEXT"`
	Website             string         `xorm:"VARCHAR(2048)"`
	OriginalServiceType GitServiceType `xorm:"index"`
	OriginalURL         string         `xorm:"VARCHAR(2048)"`
	DefaultBranch       string

	NumWatches          int
	NumStars            int
	NumForks            int
	NumIssues           int
	NumClosedIssues     int
	NumOpenIssues       int `xorm:"-"`
	NumPulls            int
	NumClosedPulls      int
	NumOpenPulls        int `xorm:"-"`
	NumMilestones       int `xorm:"NOT NULL DEFAULT 0"`
	NumClosedMilestones int `xorm:"NOT NULL DEFAULT 0"`
	NumOpenMilestones   int `xorm:"-"`
	NumProjects         int `xorm:"NOT NULL DEFAULT 0"`
	NumClosedProjects   int `xorm:"NOT NULL DEFAULT 0"`
	NumOpenProjects     int `xorm:"-"`

	IsPrivate  bool `xorm:"INDEX"`
	IsEmpty    bool `xorm:"INDEX"`
	IsArchived bool `xorm:"INDEX"`
	IsMirror   bool `xorm:"INDEX"`
	*Mirror    `xorm:"-"`
	Status     RepositoryStatus `xorm:"NOT NULL DEFAULT 0"`

	RenderingMetas         map[string]string `xorm:"-"`
	DocumentRenderingMetas map[string]string `xorm:"-"`
	Units                  []*RepoUnit       `xorm:"-"`
	PrimaryLanguage        *LanguageStat     `xorm:"-"`

	IsFork                          bool               `xorm:"INDEX NOT NULL DEFAULT false"`
	ForkID                          int64              `xorm:"INDEX"`
	BaseRepo                        *Repository        `xorm:"-"`
	IsTemplate                      bool               `xorm:"INDEX NOT NULL DEFAULT false"`
	TemplateID                      int64              `xorm:"INDEX"`
	Size                            int64              `xorm:"NOT NULL DEFAULT 0"`
	CodeIndexerStatus               *RepoIndexerStatus `xorm:"-"`
	StatsIndexerStatus              *RepoIndexerStatus `xorm:"-"`
	IsFsckEnabled                   bool               `xorm:"NOT NULL DEFAULT true"`
	CloseIssuesViaCommitInAnyBranch bool               `xorm:"NOT NULL DEFAULT false"`
	Topics                          []string           `xorm:"TEXT JSON"`

	TrustModel TrustModelType

	// Avatar: ID(10-20)-md5(32) - must fit into 64 symbols
	Avatar string `xorm:"VARCHAR(64)"`

	CreatedUnix TimeStamp `xorm:"INDEX created"`
	UpdatedUnix TimeStamp `xorm:"INDEX updated"`
}*/

// EmptySHA defines empty git SHA
const EmptySHA = "0000000000000000000000000000000000000000"

// EmptyTreeSHA is the SHA of an empty tree
const EmptyTreeSHA = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

// SHAPattern can be used to determine if a string is an valid sha
var shaPattern = regexp.MustCompile(`^[0-9a-f]{4,40}$`)

// IsValidSHAPattern will check if the provided string matches the SHA Pattern
func IsValidSHAPattern(sha string) bool {
	return shaPattern.MatchString(sha)
}

// MustID always creates a new SHA1 from a [20]byte array with no validation of input.
func MustID(b []byte) SHA1 {
	var id SHA1
	copy(id[:], b)
	return id
}

// NewID creates a new SHA1 from a [20]byte array.
func NewID(b []byte) (SHA1, error) {
	if len(b) != 20 {
		return SHA1{}, fmt.Errorf("Length must be 20: %v", b)
	}
	return MustID(b), nil
}

// NewIDFromString creates a new SHA1 from a ID string of length 40.
func NewIDFromString(s string) (SHA1, error) {
	var id SHA1
	s = strings.TrimSpace(s)
	if len(s) != 40 {
		return id, fmt.Errorf("Length must be 40: %s", s)
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return id, err
	}
	return NewID(b)
}

// Command represents a command with its subcommands or arguments.
type Command struct {
	name             string
	args             []string
	parentContext    context.Context
	desc             string
	globalArgsLength int
	brokenArgs       []string
}

var (
	// globalCommandArgs global command args for external package setting
	globalCommandArgs []CmdArg

	// defaultCommandExecutionTimeout default command execution timeout duration
	defaultCommandExecutionTimeout = 360 * time.Second

	// GitExecutable is the command name of git
	// Could be updated to an absolute path while initialization
	GitExecutable = "git"
)

// NewCommand creates and returns a new Git Command based on given command and arguments.
// Each argument should be safe to be trusted. User-provided arguments should be passed to AddDynamicArguments instead.
func NewCommand(ctx context.Context, args ...CmdArg) *Command {
	// Make an explicit copy of globalCommandArgs, otherwise append might overwrite it
	cargs := make([]string, 0, len(globalCommandArgs)+len(args))
	for _, arg := range globalCommandArgs {
		cargs = append(cargs, string(arg))
	}
	for _, arg := range args {
		cargs = append(cargs, string(arg))
	}
	return &Command{
		name:             GitExecutable,
		args:             cargs,
		parentContext:    ctx,
		globalArgsLength: len(globalCommandArgs),
	}
}

// SetDescription sets the description for this command which be returned on
// c.String()
func (c *Command) SetDescription(desc string) *Command {
	c.desc = desc
	return c
}

// RunOpts represents parameters to run the command. If UseContextTimeout is specified, then Timeout is ignored.
type RunOpts struct {
	Env               []string
	Timeout           time.Duration
	UseContextTimeout bool
	Dir               string
	Stdout, Stderr    io.Writer
	Stdin             io.Reader
	PipelineFunc      func(context.Context, context.CancelFunc) error
}

var ErrBrokenCommand = errors.New("git command is broken")

func (c *Command) String() string {
	if len(c.args) == 0 {
		return c.name
	}
	return fmt.Sprintf("%s %s", c.name, strings.Join(c.args, " "))
}

// StringToReadOnlyBytes returns bytes converted from given string.
func StringToReadOnlyBytes(s string) (bs []byte) {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&bs))
	bh.Data = sh.Data
	bh.Cap = sh.Len
	bh.Len = sh.Len
	return
}

// BytesToReadOnlyString returns a string converted from given bytes.
func BytesToReadOnlyString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

const userPlaceholder = "sanitized-credential"

var schemeSep = []byte("://")

// SanitizeCredentialURLs remove all credentials in URLs (starting with "scheme://") for the input string: "https://user:pass@domain.com" => "https://sanitized-credential@domain.com"
func SanitizeCredentialURLs(s string) string {
	bs := StringToReadOnlyBytes(s)
	schemeSepPos := bytes.Index(bs, schemeSep)
	if schemeSepPos == -1 || bytes.IndexByte(bs[schemeSepPos:], '@') == -1 {
		return s // fast return if there is no URL scheme or no userinfo
	}
	out := make([]byte, 0, len(bs)+len(userPlaceholder))
	for schemeSepPos != -1 {
		schemeSepPos += 3         // skip the "://"
		sepAtPos := -1            // the possible '@' position: "https://foo@[^here]host"
		sepEndPos := schemeSepPos // the possible end position: "The https://host[^here] in log for test"
	sepLoop:
		for ; sepEndPos < len(bs); sepEndPos++ {
			c := bs[sepEndPos]
			if ('A' <= c && c <= 'Z') || ('a' <= c && c <= 'z') || ('0' <= c && c <= '9') {
				continue
			}
			switch c {
			case '@':
				sepAtPos = sepEndPos
			case '-', '.', '_', '~', '!', '$', '&', '\'', '(', ')', '*', '+', ',', ';', '=', ':', '%':
				continue // due to RFC 3986, userinfo can contain - . _ ~ ! $ & ' ( ) * + , ; = : and any percent-encoded chars
			default:
				break sepLoop // if it is an invalid char for URL (eg: space, '/', and others), stop the loop
			}
		}
		// if there is '@', and the string is like "s://u@h", then hide the "u" part
		if sepAtPos != -1 && (schemeSepPos >= 4 && unicode.IsLetter(rune(bs[schemeSepPos-4]))) && sepAtPos-schemeSepPos > 0 && sepEndPos-sepAtPos > 0 {
			out = append(out, bs[:schemeSepPos]...)
			out = append(out, userPlaceholder...)
			out = append(out, bs[sepAtPos:sepEndPos]...)
		} else {
			out = append(out, bs[:sepEndPos]...)
		}
		bs = bs[sepEndPos:]
		schemeSepPos = bytes.Index(bs, schemeSep)
	}
	out = append(out, bs...)
	return BytesToReadOnlyString(out)
}

// IDType is a pid type
type IDType string

// FinishedFunc is a function that marks that the process is finished and can be removed from the process table
// - it is simply an alias for context.CancelFunc and is only for documentary purposes
type FinishedFunc = context.CancelFunc

// process represents a working process inheriting from Gitea.
type process struct {
	PID         IDType // Process ID, not system one.
	ParentPID   IDType
	Description string
	Start       time.Time
	Cancel      context.CancelFunc
	Type        string
}

// Manager manages all processes and counts PIDs.
type Manager struct {
	mutex sync.Mutex

	next     int64
	lastTime int64

	processMap map[IDType]*process
}

var manager *Manager
var managerInit sync.Once

// GetManager returns a Manager and initializes one as singleton if there's none yet
func GetManager() *Manager {
	managerInit.Do(func() {
		manager = &Manager{
			processMap: make(map[IDType]*process),
			next:       1,
		}
	})
	return manager
}

// Context is a wrapper around context.Context and contains the current pid for this context
type Context struct {
	context.Context
	pid IDType
}

// ProcessContextKey is the key under which process contexts are stored
var ProcessContextKey interface{} = "process-context"

// GetContext will return a process context if one exists
func GetContext(ctx context.Context) *Context {
	if pCtx, ok := ctx.(*Context); ok {
		return pCtx
	}
	pCtxInterface := ctx.Value(ProcessContextKey)
	if pCtxInterface == nil {
		return nil
	}
	if pCtx, ok := pCtxInterface.(*Context); ok {
		return pCtx
	}
	return nil
}

// GetPID returns the PID for this context
func (c *Context) GetPID() IDType {
	return c.pid
}

// GetPID returns the PID for this context
func GetPID(ctx context.Context) IDType {
	pCtx := GetContext(ctx)
	if pCtx == nil {
		return ""
	}
	return pCtx.GetPID()
}

// GetParentPID returns the ParentPID for this context
func GetParentPID(ctx context.Context) IDType {
	var parentPID IDType
	if parentProcess := GetContext(ctx); parentProcess != nil {
		parentPID = parentProcess.GetPID()
	}
	return parentPID
}

// nextPID will return the next available PID. pm.mutex should already be locked.
func (pm *Manager) nextPID() (start time.Time, pid IDType) {
	start = time.Now()
	startUnix := start.Unix()
	if pm.lastTime == startUnix {
		pm.next++
	} else {
		pm.next = 1
	}
	pm.lastTime = startUnix
	pid = IDType(strconv.FormatInt(start.Unix(), 16))

	if pm.next == 1 {
		return
	}
	pid = IDType(string(pid) + "-" + strconv.FormatInt(pm.next, 10))
	return start, pid
}

func (pm *Manager) remove(process *process) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	if p := pm.processMap[process.PID]; p == process {
		delete(pm.processMap, process.PID)
	}
}

// DescriptionPProfLabel is a label set on goroutines that have a process attached
const DescriptionPProfLabel = "process-description"

// PIDPProfLabel is a label set on goroutines that have a process attached
const PIDPProfLabel = "pid"

// PPIDPProfLabel is a label set on goroutines that have a process attached
const PPIDPProfLabel = "ppid"

// ProcessTypePProfLabel is a label set on goroutines that have a process attached
const ProcessTypePProfLabel = "process-type"

// Add create a new process
func (pm *Manager) Add(ctx context.Context, description string, cancel context.CancelFunc, processType string, currentlyRunning bool) (context.Context, IDType, FinishedFunc) {
	parentPID := GetParentPID(ctx)

	pm.mutex.Lock()
	start, pid := pm.nextPID()

	parent := pm.processMap[parentPID]
	if parent == nil {
		parentPID = ""
	}

	process := &process{
		PID:         pid,
		ParentPID:   parentPID,
		Description: description,
		Start:       start,
		Cancel:      cancel,
		Type:        processType,
	}

	var finished FinishedFunc
	if currentlyRunning {
		finished = func() {
			cancel()
			pm.remove(process)
			pprof.SetGoroutineLabels(ctx)
		}
	} else {
		finished = func() {
			cancel()
			pm.remove(process)
		}
	}

	pm.processMap[pid] = process
	pm.mutex.Unlock()

	pprofCtx := pprof.WithLabels(ctx, pprof.Labels(DescriptionPProfLabel, description, PPIDPProfLabel, string(parentPID), PIDPProfLabel, string(pid), ProcessTypePProfLabel, processType))
	if currentlyRunning {
		pprof.SetGoroutineLabels(pprofCtx)
	}

	return &Context{
		Context: pprofCtx,
		pid:     pid,
	}, pid, finished
}

var (
	SystemProcessType  = "system"
	RequestProcessType = "request"
	NormalProcessType  = "normal"
	NoneProcessType    = "none"
)

// AddContext creates a new context and adds it as a process. Once the process is finished, finished must be called
// to remove the process from the process table. It should not be called until the process is finished but must always be called.
//
// cancel should be used to cancel the returned context, however it will not remove the process from the process table.
// finished will cancel the returned context and remove it from the process table.
//
// Most processes will not need to use the cancel function but there will be cases whereby you want to cancel the process but not immediately remove it from the
// process table.
func (pm *Manager) AddContext(parent context.Context, description string) (ctx context.Context, cancel context.CancelFunc, finished FinishedFunc) {
	ctx, cancel = context.WithCancel(parent)

	ctx, _, finished = pm.Add(ctx, description, cancel, NormalProcessType, true)

	return ctx, cancel, finished
}

// AddContextTimeout creates a new context and add it as a process. Once the process is finished, finished must be called
// to remove the process from the process table. It should not be called until the process is finished but must always be called.
//
// cancel should be used to cancel the returned context, however it will not remove the process from the process table.
// finished will cancel the returned context and remove it from the process table.
//
// Most processes will not need to use the cancel function but there will be cases whereby you want to cancel the process but not immediately remove it from the
// process table.
func (pm *Manager) AddContextTimeout(parent context.Context, timeout time.Duration, description string) (ctx context.Context, cancel context.CancelFunc, finshed FinishedFunc) {
	if timeout <= 0 {
		// it's meaningless to use timeout <= 0, and it must be a bug! so we must panic here to tell developers to make the timeout correct
		panic("the timeout must be greater than zero, otherwise the context will be cancelled immediately")
	}

	ctx, cancel = context.WithTimeout(parent, timeout)

	ctx, _, finshed = pm.Add(ctx, description, cancel, NormalProcessType, true)

	return ctx, cancel, finshed
}

// SetSysProcAttribute sets the common SysProcAttrs for commands
func SetSysProcAttribute(cmd *exec.Cmd) {
	// When Gitea runs SubProcessA -> SubProcessB and SubProcessA gets killed by context timeout, use setpgid to make sure the sub processes can be reaped instead of leaving defunct(zombie) processes.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// HomeDir is the home dir for git to store the global config file used by Gitea internally
func HomeDir() string {
	//TODO
	/*if setting.Git.HomePath == "" {
		// strict check, make sure the git module is initialized correctly.
		// attention: when the git module is called in gitea sub-command (serv/hook), the log module might not obviously show messages to users/developers.
		// for example: if there is gitea git hook code calling git.NewCommand before git.InitXxx, the integration test won't show the real failure reasons.
		log.Fatal("Unable to init Git's HomeDir, incorrect initialization of the setting and git modules")
		return ""
	}*/
	return "/tmp" //TODO setting.Git.HomePath
}

func commonBaseEnvs() []string {
	// at the moment, do not set "GIT_CONFIG_NOSYSTEM", users may have put some configs like "receive.certNonceSeed" in it
	envs := []string{
		"HOME=" + HomeDir(),        // make Gitea use internal git config only, to prevent conflicts with user's git config
		"GIT_NO_REPLACE_OBJECTS=1", // ignore replace references (https://git-scm.com/docs/git-replace)
	}

	// some environment variables should be passed to git command
	passThroughEnvKeys := []string{
		"GNUPGHOME", // git may call gnupg to do commit signing
	}
	for _, key := range passThroughEnvKeys {
		if val, ok := os.LookupEnv(key); ok {
			envs = append(envs, key+"="+val)
		}
	}
	return envs
}

// DefaultLocale is the default LC_ALL to run git commands in.
const DefaultLocale = "C"

// CommonGitCmdEnvs returns the common environment variables for a "git" command.
func CommonGitCmdEnvs() []string {
	return append(commonBaseEnvs(), []string{
		"LC_ALL=" + DefaultLocale,
		"GIT_TERMINAL_PROMPT=0", // avoid prompting for credentials interactively, supported since git v2.3
	}...)
}

// Run runs the command with the RunOpts
func (c *Command) Run(opts *RunOpts) error {
	if len(c.brokenArgs) != 0 {
		//TODO log.Error("git command is broken: %s, broken args: %s", c.String(), strings.Join(c.brokenArgs, " "))
		return ErrBrokenCommand
	}
	if opts == nil {
		opts = &RunOpts{}
	}

	// We must not change the provided options
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultCommandExecutionTimeout
	}

	if len(opts.Dir) == 0 {
		//TODO log.Debug("%s", c)
	} else {
		//TODO log.Debug("%s: %v", opts.Dir, c)
	}

	desc := c.desc
	if desc == "" {
		args := c.args[c.globalArgsLength:]
		var argSensitiveURLIndexes []int
		for i, arg := range c.args {
			if strings.Contains(arg, "://") && strings.Contains(arg, "@") {
				argSensitiveURLIndexes = append(argSensitiveURLIndexes, i)
			}
		}
		if len(argSensitiveURLIndexes) > 0 {
			args = make([]string, len(c.args))
			copy(args, c.args)
			for _, urlArgIndex := range argSensitiveURLIndexes {
				args[urlArgIndex] = SanitizeCredentialURLs(args[urlArgIndex])
			}
		}
		desc = fmt.Sprintf("%s %s [repo_path: %s]", c.name, strings.Join(args, " "), opts.Dir)
	}

	var ctx context.Context
	var cancel context.CancelFunc
	var finished context.CancelFunc

	if opts.UseContextTimeout {
		ctx, cancel, finished = GetManager().AddContext(c.parentContext, desc)
	} else {
		ctx, cancel, finished = GetManager().AddContextTimeout(c.parentContext, timeout, desc)
	}
	defer finished()

	cmd := exec.CommandContext(ctx, c.name, c.args...)
	if opts.Env == nil {
		cmd.Env = os.Environ()
	} else {
		cmd.Env = opts.Env
	}

	SetSysProcAttribute(cmd)
	cmd.Env = append(cmd.Env, CommonGitCmdEnvs()...)
	cmd.Dir = opts.Dir
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr
	cmd.Stdin = opts.Stdin
	if err := cmd.Start(); err != nil {
		return err
	}

	if opts.PipelineFunc != nil {
		err := opts.PipelineFunc(ctx, cancel)
		if err != nil {
			cancel()
			_ = cmd.Wait()
			return err
		}
	}

	if err := cmd.Wait(); err != nil && ctx.Err() != context.DeadlineExceeded {
		return err
	}

	return ctx.Err()
}

var callerPrefix string

// ConcatenateError concatenats an error with stderr string
func ConcatenateError(err error, stderr string) error {
	if len(stderr) == 0 {
		return err
	}
	return fmt.Errorf("%w - %s", err, stderr)
}

// CatFileBatchCheck opens git cat-file --batch-check in the provided repo and returns a stdin pipe, a stdout reader and cancel function
func CatFileBatchCheck(ctx context.Context, repoPath string) (WriteCloserError, *bufio.Reader, func()) {
	batchStdinReader, batchStdinWriter := io.Pipe()
	batchStdoutReader, batchStdoutWriter := io.Pipe()
	ctx, ctxCancel := context.WithCancel(ctx)
	closed := make(chan struct{})
	cancel := func() {
		ctxCancel()
		_ = batchStdoutReader.Close()
		_ = batchStdinWriter.Close()
		<-closed
	}

	// Ensure cancel is called as soon as the provided context is cancelled
	go func() {
		<-ctx.Done()
		cancel()
	}()

	_, filename, line, _ := runtime.Caller(2)
	filename = strings.TrimPrefix(filename, callerPrefix)

	go func() {
		stderr := strings.Builder{}
		err := NewCommand(ctx, "cat-file", "--batch-check").
			SetDescription(fmt.Sprintf("%s cat-file --batch-check [repo_path: %s] (%s:%d)", GitExecutable, repoPath, filename, line)).
			Run(&RunOpts{
				Dir:    repoPath,
				Stdin:  batchStdinReader,
				Stdout: batchStdoutWriter,
				Stderr: &stderr,
			})
		if err != nil {
			_ = batchStdoutWriter.CloseWithError(ConcatenateError(err, (&stderr).String()))
			_ = batchStdinReader.CloseWithError(ConcatenateError(err, (&stderr).String()))
		} else {
			_ = batchStdoutWriter.Close()
			_ = batchStdinReader.Close()
		}
		close(closed)
	}()

	// For simplicities sake we'll use a buffered reader to read from the cat-file --batch-check
	batchReader := bufio.NewReader(batchStdoutReader)

	return batchStdinWriter, batchReader, cancel
}

// CatFileBatchCheck obtains a CatFileBatchCheck for this repository
func (repo *Repository) CatFileBatchCheck(ctx context.Context) (WriteCloserError, *bufio.Reader, func()) {
	if repo.checkCancel == nil || repo.checkReader.Buffered() > 0 {
		//TODO log.Debug("Opening temporary cat file batch-check: %s", repo.Path)
		return CatFileBatchCheck(ctx, repo.Path)
	}
	return repo.checkWriter, repo.checkReader, func() {}
}

// ErrExecTimeout error when exec timed out
type ErrExecTimeout struct {
	Duration time.Duration
}

// IsErrExecTimeout if some error is ErrExecTimeout
func IsErrExecTimeout(err error) bool {
	_, ok := err.(ErrExecTimeout)
	return ok
}

func (err ErrExecTimeout) Error() string {
	return fmt.Sprintf("execution is timeout [duration: %v]", err.Duration)
}

// Common Errors forming the base of our error system
//
// Many Errors returned by Gitea can be tested against these errors
// using errors.Is.
var (
	ErrInvalidArgument  = errors.New("invalid argument")
	ErrPermissionDenied = errors.New("permission denied")
	ErrAlreadyExist     = errors.New("resource already exists")
	EErrNotExist        = errors.New("resource does not exist")
)

// ErrNotExist commit not exist error
type ErrNotExist struct {
	ID      string
	RelPath string
}

// IsErrNotExist if some error is ErrNotExist
func IsErrNotExist(err error) bool {
	_, ok := err.(ErrNotExist)
	return ok
}

func (err ErrNotExist) Error() string {
	return fmt.Sprintf("object does not exist [id: %s, rel_path: %s]", err.ID, err.RelPath)
}

func (err ErrNotExist) Unwrap() error {
	return EErrNotExist
}

// ErrBadLink entry.FollowLink error
type ErrBadLink struct {
	Name    string
	Message string
}

func (err ErrBadLink) Error() string {
	return fmt.Sprintf("%s: %s", err.Name, err.Message)
}

// IsErrBadLink if some error is ErrBadLink
func IsErrBadLink(err error) bool {
	_, ok := err.(ErrBadLink)
	return ok
}

// ErrUnsupportedVersion error when required git version not matched
type ErrUnsupportedVersion struct {
	Required string
}

// IsErrUnsupportedVersion if some error is ErrUnsupportedVersion
func IsErrUnsupportedVersion(err error) bool {
	_, ok := err.(ErrUnsupportedVersion)
	return ok
}

func (err ErrUnsupportedVersion) Error() string {
	return fmt.Sprintf("Operation requires higher version [required: %s]", err.Required)
}

// ErrBranchNotExist represents a "BranchNotExist" kind of error.
type ErrBranchNotExist struct {
	Name string
}

// IsErrBranchNotExist checks if an error is a ErrBranchNotExist.
func IsErrBranchNotExist(err error) bool {
	_, ok := err.(ErrBranchNotExist)
	return ok
}

func (err ErrBranchNotExist) Error() string {
	return fmt.Sprintf("branch does not exist [name: %s]", err.Name)
}

func (err ErrBranchNotExist) Unwrap() error {
	return EErrNotExist
}

// ErrPushOutOfDate represents an error if merging fails due to unrelated histories
type ErrPushOutOfDate struct {
	StdOut string
	StdErr string
	Err    error
}

// IsErrPushOutOfDate checks if an error is a ErrPushOutOfDate.
func IsErrPushOutOfDate(err error) bool {
	_, ok := err.(*ErrPushOutOfDate)
	return ok
}

func (err *ErrPushOutOfDate) Error() string {
	return fmt.Sprintf("PushOutOfDate Error: %v: %s\n%s", err.Err, err.StdErr, err.StdOut)
}

// Unwrap unwraps the underlying error
func (err *ErrPushOutOfDate) Unwrap() error {
	return fmt.Errorf("%w - %s", err.Err, err.StdErr)
}

// ErrPushRejected represents an error if merging fails due to rejection from a hook
type ErrPushRejected struct {
	Message string
	StdOut  string
	StdErr  string
	Err     error
}

// IsErrPushRejected checks if an error is a ErrPushRejected.
func IsErrPushRejected(err error) bool {
	_, ok := err.(*ErrPushRejected)
	return ok
}

func (err *ErrPushRejected) Error() string {
	return fmt.Sprintf("PushRejected Error: %v: %s\n%s", err.Err, err.StdErr, err.StdOut)
}

// Unwrap unwraps the underlying error
func (err *ErrPushRejected) Unwrap() error {
	return fmt.Errorf("%w - %s", err.Err, err.StdErr)
}

// GenerateMessage generates the remote message from the stderr
func (err *ErrPushRejected) GenerateMessage() {
	messageBuilder := &strings.Builder{}
	i := strings.Index(err.StdErr, "remote: ")
	if i < 0 {
		err.Message = ""
		return
	}
	for {
		if len(err.StdErr) <= i+8 {
			break
		}
		if err.StdErr[i:i+8] != "remote: " {
			break
		}
		i += 8
		nl := strings.IndexByte(err.StdErr[i:], '\n')
		if nl >= 0 {
			messageBuilder.WriteString(err.StdErr[i : i+nl+1])
			i = i + nl + 1
		} else {
			messageBuilder.WriteString(err.StdErr[i:])
			i = len(err.StdErr)
		}
	}
	err.Message = strings.TrimSpace(messageBuilder.String())
}

// ErrMoreThanOne represents an error if pull request fails when there are more than one sources (branch, tag) with the same name
type ErrMoreThanOne struct {
	StdOut string
	StdErr string
	Err    error
}

// IsErrMoreThanOne checks if an error is a ErrMoreThanOne
func IsErrMoreThanOne(err error) bool {
	_, ok := err.(*ErrMoreThanOne)
	return ok
}

func (err *ErrMoreThanOne) Error() string {
	return fmt.Sprintf("ErrMoreThanOne Error: %v: %s\n%s", err.Err, err.StdErr, err.StdOut)
}

// ReadBatchLine reads the header line from cat-file --batch
// We expect:
// <sha> SP <type> SP <size> LF
// sha is a 40byte not 20byte here
func ReadBatchLine(rd *bufio.Reader) (sha []byte, typ string, size int64, err error) {
	typ, err = rd.ReadString('\n')
	if err != nil {
		return
	}
	if len(typ) == 1 {
		typ, err = rd.ReadString('\n')
		if err != nil {
			return
		}
	}
	idx := strings.IndexByte(typ, ' ')
	if idx < 0 {
		//TODO log.Debug("missing space typ: %s", typ)
		err = ErrNotExist{ID: string(sha)}
		return
	}
	sha = []byte(typ[:idx])
	typ = typ[idx+1:]

	idx = strings.IndexByte(typ, ' ')
	if idx < 0 {
		err = ErrNotExist{ID: string(sha)}
		return
	}

	sizeStr := typ[idx+1 : len(typ)-1]
	typ = typ[:idx]

	size, err = strconv.ParseInt(sizeStr, 10, 64)
	return sha, typ, size, err
}

// MustIDFromString always creates a new sha from a ID with no validation of input.
func MustIDFromString(s string) SHA1 {
	b, _ := hex.DecodeString(s)
	return MustID(b)
}

// ConvertToSHA1 returns a Hash object from a potential ID string
func (repo *Repository) ConvertToSHA1(commitID string) (SHA1, error) {
	if len(commitID) == 40 && IsValidSHAPattern(commitID) {
		sha1, err := NewIDFromString(commitID)
		if err == nil {
			return sha1, nil
		}
	}

	wr, rd, cancel := repo.CatFileBatchCheck(repo.Ctx)
	defer cancel()
	_, err := wr.Write([]byte(commitID + "\n"))
	if err != nil {
		return SHA1{}, err
	}
	sha, _, _, err := ReadBatchLine(rd)
	if err != nil {
		if IsErrNotExist(err) {
			return SHA1{}, ErrNotExist{commitID, ""}
		}
		return SHA1{}, err
	}

	return MustIDFromString(string(sha)), nil
}

const (
	// GitTimeLayout is the (default) time layout used by git.
	GitTimeLayout = "Mon Jan _2 15:04:05 2006 -0700"
)

// Helper to get a signature from the commit line, which looks like these:
//
//	author Patrick Gundlach <gundlach@speedata.de> 1378823654 +0200
//	author Patrick Gundlach <gundlach@speedata.de> Thu, 07 Apr 2005 22:13:13 +0200
//
// but without the "author " at the beginning (this method should)
// be used for author and committer.
func newSignatureFromCommitline(line []byte) (sig *Signature, err error) {
	sig = new(Signature)
	emailStart := bytes.LastIndexByte(line, '<')
	emailEnd := bytes.LastIndexByte(line, '>')
	if emailStart == -1 || emailEnd == -1 || emailEnd < emailStart {
		return
	}

	if emailStart > 0 { // Empty name has already occurred, even if it shouldn't
		sig.Name = strings.TrimSpace(string(line[:emailStart-1]))
	}
	sig.Email = string(line[emailStart+1 : emailEnd])

	hasTime := emailEnd+2 < len(line)
	if !hasTime {
		return
	}

	// Check date format.
	firstChar := line[emailEnd+2]
	if firstChar >= 48 && firstChar <= 57 {
		idx := bytes.IndexByte(line[emailEnd+2:], ' ')
		if idx < 0 {
			return
		}

		timestring := string(line[emailEnd+2 : emailEnd+2+idx])
		seconds, _ := strconv.ParseInt(timestring, 10, 64)
		sig.When = time.Unix(seconds, 0)

		idx += emailEnd + 3
		if idx >= len(line) || idx+5 > len(line) {
			return
		}

		timezone := string(line[idx : idx+5])
		tzhours, err1 := strconv.ParseInt(timezone[0:3], 10, 64)
		tzmins, err2 := strconv.ParseInt(timezone[3:], 10, 64)
		if err1 != nil || err2 != nil {
			return
		}
		if tzhours < 0 {
			tzmins *= -1
		}
		tz := time.FixedZone("", int(tzhours*60*60+tzmins*60))
		sig.When = sig.When.In(tz)
	} else {
		sig.When, err = time.Parse(GitTimeLayout, string(line[emailEnd+2:]))
		if err != nil {
			return
		}
	}
	return sig, err
}

// Tag represents a Git tag.
type Tag struct {
	Name      string
	ID        SHA1
	Object    SHA1 // The id of this commit object
	Type      string
	Tagger    *Signature
	Message   string
	Signature *CommitGPGSignature
}

const (
	beginpgp = "\n-----BEGIN PGP SIGNATURE-----\n"
	endpgp   = "\n-----END PGP SIGNATURE-----"
)

// Parse commit information from the (uncompressed) raw
// data from the commit object.
// \n\n separate headers from message
func parseTagData(data []byte) (*Tag, error) {
	tag := new(Tag)
	tag.Tagger = &Signature{}
	// we now have the contents of the commit object. Let's investigate...
	nextline := 0
l:
	for {
		eol := bytes.IndexByte(data[nextline:], '\n')
		switch {
		case eol > 0:
			line := data[nextline : nextline+eol]
			spacepos := bytes.IndexByte(line, ' ')
			reftype := line[:spacepos]
			switch string(reftype) {
			case "object":
				id, err := NewIDFromString(string(line[spacepos+1:]))
				if err != nil {
					return nil, err
				}
				tag.Object = id
			case "type":
				// A commit can have one or more parents
				tag.Type = string(line[spacepos+1:])
			case "tagger":
				sig, err := newSignatureFromCommitline(line[spacepos+1:])
				if err != nil {
					return nil, err
				}
				tag.Tagger = sig
			}
			nextline += eol + 1
		case eol == 0:
			tag.Message = string(data[nextline+1:])
			break l
		default:
			break l
		}
	}
	idx := strings.LastIndex(tag.Message, beginpgp)
	if idx > 0 {
		endSigIdx := strings.Index(tag.Message[idx:], endpgp)
		if endSigIdx > 0 {
			tag.Signature = &CommitGPGSignature{
				Signature: tag.Message[idx+1 : idx+endSigIdx+len(endpgp)],
				Payload:   string(data[:bytes.LastIndex(data, []byte(beginpgp))+1]),
			}
			tag.Message = tag.Message[:idx+1]
		}
	}
	return tag, nil
}

// Commit return the commit of the tag reference
func (tag *Tag) Commit(gitRepo *Repository) (*Commit, error) {
	return gitRepo.getCommit(tag.Object)
}

// NewTree create a new tree according the repository and tree id
func NewTree(repo *Repository, id SHA1) *Tree {
	return &Tree{
		ID:   id,
		repo: repo,
	}
}

// Decode decodes a byte array representing a signature to signature
func (s *Signature) Decode(b []byte) {
	sig, _ := newSignatureFromCommitline(b)
	s.Email = sig.Email
	s.Name = sig.Name
	s.When = sig.When
}

// CommitFromReader will generate a Commit from a provided reader
// We need this to interpret commits from cat-file or cat-file --batch
//
// If used as part of a cat-file --batch stream you need to limit the reader to the correct size
func CommitFromReader(gitRepo *Repository, sha SHA1, reader io.Reader) (*Commit, error) {
	commit := &Commit{
		ID:        sha,
		Author:    &Signature{},
		Committer: &Signature{},
	}

	payloadSB := new(strings.Builder)
	signatureSB := new(strings.Builder)
	messageSB := new(strings.Builder)
	message := false
	pgpsig := false

	bufReader, ok := reader.(*bufio.Reader)
	if !ok {
		bufReader = bufio.NewReader(reader)
	}

readLoop:
	for {
		line, err := bufReader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				if message {
					_, _ = messageSB.Write(line)
				}
				_, _ = payloadSB.Write(line)
				break readLoop
			}
			return nil, err
		}
		if pgpsig {
			if len(line) > 0 && line[0] == ' ' {
				_, _ = signatureSB.Write(line[1:])
				continue
			} else {
				pgpsig = false
			}
		}

		if !message {
			// This is probably not correct but is copied from go-gits interpretation...
			trimmed := bytes.TrimSpace(line)
			if len(trimmed) == 0 {
				message = true
				_, _ = payloadSB.Write(line)
				continue
			}

			split := bytes.SplitN(trimmed, []byte{' '}, 2)
			var data []byte
			if len(split) > 1 {
				data = split[1]
			}

			switch string(split[0]) {
			case "tree":
				commit.Tree = *NewTree(gitRepo, MustIDFromString(string(data)))
				_, _ = payloadSB.Write(line)
			case "parent":
				commit.Parents = append(commit.Parents, MustIDFromString(string(data)))
				_, _ = payloadSB.Write(line)
			case "author":
				commit.Author = &Signature{}
				commit.Author.Decode(data)
				_, _ = payloadSB.Write(line)
			case "committer":
				commit.Committer = &Signature{}
				commit.Committer.Decode(data)
				_, _ = payloadSB.Write(line)
			case "gpgsig":
				_, _ = signatureSB.Write(data)
				_ = signatureSB.WriteByte('\n')
				pgpsig = true
			}
		} else {
			_, _ = messageSB.Write(line)
			_, _ = payloadSB.Write(line)
		}
	}
	commit.CommitMessage = messageSB.String()
	commit.Signature = &CommitGPGSignature{
		Signature: signatureSB.String(),
		Payload:   payloadSB.String(),
	}
	if len(commit.Signature.Signature) == 0 {
		commit.Signature = nil
	}

	return commit, nil
}

func (repo *Repository) getCommitFromBatchReader(rd *bufio.Reader, id SHA1) (*Commit, error) {
	_, typ, size, err := ReadBatchLine(rd)
	if err != nil {
		if errors.Is(err, io.EOF) || IsErrNotExist(err) {
			return nil, ErrNotExist{ID: id.String()}
		}
		return nil, err
	}

	switch typ {
	case "missing":
		return nil, ErrNotExist{ID: id.String()}
	case "tag":
		// then we need to parse the tag
		// and load the commit
		data, err := io.ReadAll(io.LimitReader(rd, size))
		if err != nil {
			return nil, err
		}
		_, err = rd.Discard(1)
		if err != nil {
			return nil, err
		}
		tag, err := parseTagData(data)
		if err != nil {
			return nil, err
		}

		commit, err := tag.Commit(repo)
		if err != nil {
			return nil, err
		}

		commit.CommitMessage = strings.TrimSpace(tag.Message)
		commit.Author = tag.Tagger
		commit.Signature = tag.Signature

		return commit, nil
	case "commit":
		commit, err := CommitFromReader(repo, id, io.LimitReader(rd, size))
		if err != nil {
			return nil, err
		}
		_, err = rd.Discard(1)
		if err != nil {
			return nil, err
		}

		return commit, nil
	default:
		//TODO log.Debug("Unknown typ: %s", typ)
		_, err = rd.Discard(int(size) + 1)
		if err != nil {
			return nil, err
		}
		return nil, ErrNotExist{
			ID: id.String(),
		}
	}
}

// String returns a string representation of the SHA
func (s SHA1) String() string {
	return hex.EncodeToString(s[:])
}

// CatFileBatch opens git cat-file --batch in the provided repo and returns a stdin pipe, a stdout reader and cancel function
func CatFileBatch(ctx context.Context, repoPath string) (WriteCloserError, *bufio.Reader, func()) {
	// We often want to feed the commits in order into cat-file --batch, followed by their trees and sub trees as necessary.
	// so let's create a batch stdin and stdout
	batchStdinReader, batchStdinWriter := io.Pipe()
	batchStdoutReader, batchStdoutWriter := nio.Pipe(buffer.New(32 * 1024))
	ctx, ctxCancel := context.WithCancel(ctx)
	closed := make(chan struct{})
	cancel := func() {
		ctxCancel()
		_ = batchStdinWriter.Close()
		_ = batchStdoutReader.Close()
		<-closed
	}

	// Ensure cancel is called as soon as the provided context is cancelled
	go func() {
		<-ctx.Done()
		cancel()
	}()

	_, filename, line, _ := runtime.Caller(2)
	filename = strings.TrimPrefix(filename, callerPrefix)

	go func() {
		stderr := strings.Builder{}
		err := NewCommand(ctx, "cat-file", "--batch").
			SetDescription(fmt.Sprintf("%s cat-file --batch [repo_path: %s] (%s:%d)", GitExecutable, repoPath, filename, line)).
			Run(&RunOpts{
				Dir:    repoPath,
				Stdin:  batchStdinReader,
				Stdout: batchStdoutWriter,
				Stderr: &stderr,
			})
		if err != nil {
			_ = batchStdoutWriter.CloseWithError(ConcatenateError(err, (&stderr).String()))
			_ = batchStdinReader.CloseWithError(ConcatenateError(err, (&stderr).String()))
		} else {
			_ = batchStdoutWriter.Close()
			_ = batchStdinReader.Close()
		}
		close(closed)
	}()

	// For simplicities sake we'll us a buffered reader to read from the cat-file --batch
	batchReader := bufio.NewReaderSize(batchStdoutReader, 32*1024)

	return batchStdinWriter, batchReader, cancel
}

// CatFileBatch obtains a CatFileBatch for this repository
func (repo *Repository) CatFileBatch(ctx context.Context) (WriteCloserError, *bufio.Reader, func()) {
	if repo.batchCancel == nil || repo.batchReader.Buffered() > 0 {
		//TODO log.Debug("Opening temporary cat file batch for: %s", repo.Path)
		return CatFileBatch(ctx, repo.Path)
	}
	return repo.batchWriter, repo.batchReader, func() {}
}

func (repo *Repository) getCommit(id SHA1) (*Commit, error) {
	wr, rd, cancel := repo.CatFileBatch(repo.Ctx)
	defer cancel()

	_, _ = wr.Write([]byte(id.String() + "\n"))

	return repo.getCommitFromBatchReader(rd, id)
}

// GetCommit returns commit object of by ID string.
func (repo *Repository) GetCommit(commitID string) (*Commit, error) {
	id, err := repo.ConvertToSHA1(commitID)
	if err != nil {
		return nil, err
	}

	return repo.getCommit(id)
}

// ParentCount returns number of parents of the commit.
// 0 if this is the root commit,  otherwise 1,2, etc.
func (c *Commit) ParentCount() int {
	return len(c.Parents)
}

// CmdArgCheck checks whether the string is safe to be used as a dynamic argument.
// It panics if the check fails. Usually it should not be used, it's just for refactoring purpose
// deprecated
func CmdArgCheck(s string) CmdArg {
	if s != "" && s[0] == '-' {
		panic("invalid git cmd argument: " + s)
	}
	return CmdArg(s)
}

// ParentID returns oid of n-th parent (0-based index).
// It returns nil if no such parent exists.
func (c *Commit) ParentID(n int) (SHA1, error) {
	if n >= len(c.Parents) {
		return SHA1{}, ErrNotExist{"", ""}
	}
	return c.Parents[n], nil
}

// Parent returns n-th parent (0-based index) of the commit.
func (c *Commit) Parent(n int) (*Commit, error) {
	id, err := c.ParentID(n)
	if err != nil {
		return nil, err
	}
	parent, err := c.repo.getCommit(id)
	if err != nil {
		return nil, err
	}
	return parent, nil
}

// Version represents a single version.
type Version struct {
	metadata string
	pre      string
	segments []int64
	si       int
	original string
}

// The compiled regular expression used to test the validity of a version.
var (
	versionRegexp *regexp.Regexp
	semverRegexp  *regexp.Regexp
)

// The raw regular expression string used for testing the validity
// of a version.
const (
	VersionRegexpRaw string = `(?:\w+\-)*[vV]?` + // Optional prefixes, will be ignored for parsing
		`([0-9]+(\.[0-9]+)*?)` + // ( MajorNum ( '.' MinorNums ) *? )
		`(-` + // Followed by (optionally): ( '-'
		`([0-9]+[0-9A-Za-z\-~]*(\.[0-9A-Za-z\-~]+)*)` + // Either ( PreNum String ( '.' OtherString ) * )
		`|` +
		`([-\.]?([A-Za-z\-~]+[0-9A-Za-z\-~]*(\.[0-9A-Za-z\-~]+)*)))?` + // Or ( ['-' '.' ] ? ( AlphaHyphenTilde String * ( '.' String ) *  ))) ?
		`(\+([0-9A-Za-z\-~]+(\.[0-9A-Za-z\-~]+)*))?` + // and more Optionally: ( '+' String ( '.' String ) * )
		`([\+\.\-~]g[0-9A-Fa-f]{10}$)?` + // Optionally a: ( Punct 'g' Sha )
		`?`

	// SemverRegexpRaw requires a separator between version and prerelease
	SemverRegexpRaw string = `v?([0-9]+(\.[0-9]+)*?)` +
		`(-([0-9]+[0-9A-Za-z\-~]*(\.[0-9A-Za-z\-~]+)*)|(-([A-Za-z\-~]+[0-9A-Za-z\-~]*(\.[0-9A-Za-z\-~]+)*)))?` +
		`(\+([0-9A-Za-z\-~]+(\.[0-9A-Za-z\-~]+)*))?` +
		`?`
)

func init() {
	versionRegexp = regexp.MustCompile("^" + VersionRegexpRaw + "$")
	semverRegexp = regexp.MustCompile("^" + SemverRegexpRaw + "$")
}

func newVersion(v string, pattern *regexp.Regexp) (*Version, error) {
	matches := pattern.FindStringSubmatch(v)
	if matches == nil {
		return nil, fmt.Errorf("Malformed version: %s", v)
	}
	segmentsStr := strings.Split(matches[1], ".")
	segments := make([]int64, len(segmentsStr))
	si := 0
	for i, str := range segmentsStr {
		val, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return nil, fmt.Errorf(
				"Error parsing version: %s", err)
		}

		segments[i] = int64(val)
		si++
	}

	// Even though we could support more than three segments, if we
	// got less than three, pad it with 0s. This is to cover the basic
	// default usecase of semver, which is MAJOR.MINOR.PATCH at the minimum
	for i := len(segments); i < 3; i++ {
		segments = append(segments, 0)
	}

	pre := matches[7]
	if pre == "" {
		pre = matches[4]
	}

	return &Version{
		metadata: matches[10],
		pre:      pre,
		segments: segments,
		si:       si,
		original: v,
	}, nil
}

// NewVersion parses the given version and returns a new
// Version.
func NewVersion(v string) (*Version, error) {
	return newVersion(v, versionRegexp)
}

var gitVersion *Version

// DefaultContext is the default context to run git commands in, must be initialized by git.InitXxx
var DefaultContext context.Context

func bytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b)) // that's what Golang's strings.Builder.String() does (go/src/strings/builder.go)
}

type runStdError struct {
	err    error
	stderr string
	errMsg string
}

type RunStdError interface {
	error
	Unwrap() error
	Stderr() string
	IsExitCode(code int) bool
}

func (r *runStdError) Error() string {
	// the stderr must be in the returned error text, some code only checks `strings.Contains(err.Error(), "git error")`
	if r.errMsg == "" {
		r.errMsg = ConcatenateError(r.err, r.stderr).Error()
	}
	return r.errMsg
}

func (r *runStdError) Unwrap() error {
	return r.err
}

func (r *runStdError) Stderr() string {
	return r.stderr
}

func (r *runStdError) IsExitCode(code int) bool {
	var exitError *exec.ExitError
	if errors.As(r.err, &exitError) {
		return exitError.ExitCode() == code
	}
	return false
}

// RunStdBytes runs the command with options and returns stdout/stderr as bytes. and store stderr to returned error (err combined with stderr).
func (c *Command) RunStdBytes(opts *RunOpts) (stdout, stderr []byte, runErr RunStdError) {
	if opts == nil {
		opts = &RunOpts{}
	}
	if opts.Stdout != nil || opts.Stderr != nil {
		// we must panic here, otherwise there would be bugs if developers set Stdin/Stderr by mistake, and it would be very difficult to debug
		panic("stdout and stderr field must be nil when using RunStdBytes")
	}
	stdoutBuf := &bytes.Buffer{}
	stderrBuf := &bytes.Buffer{}

	// We must not change the provided options as it could break future calls - therefore make a copy.
	newOpts := &RunOpts{
		Env:               opts.Env,
		Timeout:           opts.Timeout,
		UseContextTimeout: opts.UseContextTimeout,
		Dir:               opts.Dir,
		Stdout:            stdoutBuf,
		Stderr:            stderrBuf,
		Stdin:             opts.Stdin,
		PipelineFunc:      opts.PipelineFunc,
	}

	err := c.Run(newOpts)
	stderr = stderrBuf.Bytes()
	if err != nil {
		return nil, stderr, &runStdError{err: err, stderr: bytesToString(stderr)}
	}
	// even if there is no err, there could still be some stderr output
	return stdoutBuf.Bytes(), stderr, nil
}

// RunStdString runs the command with options and returns stdout/stderr as string. and store stderr to returned error (err combined with stderr).
func (c *Command) RunStdString(opts *RunOpts) (stdout, stderr string, runErr RunStdError) {
	stdoutBytes, stderrBytes, err := c.RunStdBytes(opts)
	stdout = bytesToString(stdoutBytes)
	stderr = bytesToString(stderrBytes)
	if err != nil {
		return stdout, stderr, &runStdError{err: err, stderr: stderr}
	}
	// even if there is no err, there could still be some stderr output, so we just return stdout/stderr as they are
	return stdout, stderr, nil
}

// loadGitVersion returns current Git version from shell. Internal usage only.
func loadGitVersion() (*Version, error) {
	// doesn't need RWMutex because it's executed by Init()
	if gitVersion != nil {
		return gitVersion, nil
	}

	stdout, _, runErr := NewCommand(DefaultContext, "version").RunStdString(nil)
	if runErr != nil {
		return nil, runErr
	}

	fields := strings.Fields(stdout)
	if len(fields) < 3 {
		return nil, fmt.Errorf("invalid git version output: %s", stdout)
	}

	var versionString string

	// Handle special case on Windows.
	i := strings.Index(fields[2], "windows")
	if i >= 1 {
		versionString = fields[2][:i-1]
	} else {
		versionString = fields[2]
	}

	var err error
	gitVersion, err = NewVersion(versionString)
	return gitVersion, err
}

// String returns the full version string included pre-release
// and metadata information.
//
// This value is rebuilt according to the parsed segments and other
// information. Therefore, ambiguities in the version string such as
// prefixed zeroes (1.04.0 => 1.4.0), `v` prefix (v1.0.0 => 1.0.0), and
// missing parts (1.0 => 1.0.0) will be made into a canonicalized form
// as shown in the parenthesized examples.
func (v *Version) String() string {
	var buf bytes.Buffer
	fmtParts := make([]string, len(v.segments))
	for i, s := range v.segments {
		// We can ignore err here since we've pre-parsed the values in segments
		str := strconv.FormatInt(s, 10)
		fmtParts[i] = str
	}
	fmt.Fprint(&buf, strings.Join(fmtParts, "."))
	if v.pre != "" {
		fmt.Fprintf(&buf, "-%s", v.pre)
	}
	if v.metadata != "" {
		fmt.Fprintf(&buf, "+%s", v.metadata)
	}

	return buf.String()
}

// Segments64 returns the numeric segments of the version as a slice of int64s.
//
// This excludes any metadata or pre-release information. For example,
// for a version "1.2.3-beta", segments will return a slice of
// 1, 2, 3.
func (v *Version) Segments64() []int64 {
	result := make([]int64, len(v.segments))
	copy(result, v.segments)
	return result
}

// Prerelease returns any prerelease data that is part of the version,
// or blank if there is no prerelease data.
//
// Prerelease information is anything that comes after the "-" in the
// version (but before any metadata). For example, with "1.2.3-beta",
// the prerelease information is "beta".
func (v *Version) Prerelease() string {
	return v.pre
}

func comparePart(preSelf string, preOther string) int {
	if preSelf == preOther {
		return 0
	}

	var selfInt int64
	selfNumeric := true
	selfInt, err := strconv.ParseInt(preSelf, 10, 64)
	if err != nil {
		selfNumeric = false
	}

	var otherInt int64
	otherNumeric := true
	otherInt, err = strconv.ParseInt(preOther, 10, 64)
	if err != nil {
		otherNumeric = false
	}

	// if a part is empty, we use the other to decide
	if preSelf == "" {
		if otherNumeric {
			return -1
		}
		return 1
	}

	if preOther == "" {
		if selfNumeric {
			return 1
		}
		return -1
	}

	if selfNumeric && !otherNumeric {
		return -1
	} else if !selfNumeric && otherNumeric {
		return 1
	} else if !selfNumeric && !otherNumeric && preSelf > preOther {
		return 1
	} else if selfInt > otherInt {
		return 1
	}

	return -1
}

func comparePrereleases(v string, other string) int {
	// the same pre release!
	if v == other {
		return 0
	}

	// split both pre releases for analyse their parts
	selfPreReleaseMeta := strings.Split(v, ".")
	otherPreReleaseMeta := strings.Split(other, ".")

	selfPreReleaseLen := len(selfPreReleaseMeta)
	otherPreReleaseLen := len(otherPreReleaseMeta)

	biggestLen := otherPreReleaseLen
	if selfPreReleaseLen > otherPreReleaseLen {
		biggestLen = selfPreReleaseLen
	}

	// loop for parts to find the first difference
	for i := 0; i < biggestLen; i++ {
		partSelfPre := ""
		if i < selfPreReleaseLen {
			partSelfPre = selfPreReleaseMeta[i]
		}

		partOtherPre := ""
		if i < otherPreReleaseLen {
			partOtherPre = otherPreReleaseMeta[i]
		}

		compare := comparePart(partSelfPre, partOtherPre)
		// if parts are equals, continue the loop
		if compare != 0 {
			return compare
		}
	}

	return 0
}

func allZero(segs []int64) bool {
	for _, s := range segs {
		if s != 0 {
			return false
		}
	}
	return true
}

// Compare compares this version to another version. This
// returns -1, 0, or 1 if this version is smaller, equal,
// or larger than the other version, respectively.
//
// If you want boolean results, use the LessThan, Equal,
// GreaterThan, GreaterThanOrEqual or LessThanOrEqual methods.
func (v *Version) Compare(other *Version) int {
	// A quick, efficient equality check
	if v.String() == other.String() {
		return 0
	}

	segmentsSelf := v.Segments64()
	segmentsOther := other.Segments64()

	// If the segments are the same, we must compare on prerelease info
	if reflect.DeepEqual(segmentsSelf, segmentsOther) {
		preSelf := v.Prerelease()
		preOther := other.Prerelease()
		if preSelf == "" && preOther == "" {
			return 0
		}
		if preSelf == "" {
			return 1
		}
		if preOther == "" {
			return -1
		}

		return comparePrereleases(preSelf, preOther)
	}

	// Get the highest specificity (hS), or if they're equal, just use segmentSelf length
	lenSelf := len(segmentsSelf)
	lenOther := len(segmentsOther)
	hS := lenSelf
	if lenSelf < lenOther {
		hS = lenOther
	}
	// Compare the segments
	// Because a constraint could have more/less specificity than the version it's
	// checking, we need to account for a lopsided or jagged comparison
	for i := 0; i < hS; i++ {
		if i > lenSelf-1 {
			// This means Self had the lower specificity
			// Check to see if the remaining segments in Other are all zeros
			if !allZero(segmentsOther[i:]) {
				// if not, it means that Other has to be greater than Self
				return -1
			}
			break
		} else if i > lenOther-1 {
			// this means Other had the lower specificity
			// Check to see if the remaining segments in Self are all zeros -
			if !allZero(segmentsSelf[i:]) {
				// if not, it means that Self has to be greater than Other
				return 1
			}
			break
		}
		lhs := segmentsSelf[i]
		rhs := segmentsOther[i]
		if lhs == rhs {
			continue
		} else if lhs < rhs {
			return -1
		}
		// Otherwis, rhs was > lhs, they're not equal
		return 1
	}

	// if we got this far, they're equal
	return 0
}

// Original returns the original parsed version as-is, including any
// potential whitespace, `v` prefix, etc.
func (v *Version) Original() string {
	return v.original
}

// CheckGitVersionAtLeast check git version is at least the constraint version
func CheckGitVersionAtLeast(atLeast string) error {
	if _, err := loadGitVersion(); err != nil {
		return err
	}
	atLeastVersion, err := NewVersion(atLeast)
	if err != nil {
		return err
	}
	if gitVersion.Compare(atLeastVersion) < 0 {
		return fmt.Errorf("installed git binary version %s is not at least %s", gitVersion.Original(), atLeast)
	}
	return nil
}

func readFileName(rd *strings.Reader) (string, bool) {
	ambiguity := false
	var name string
	char, _ := rd.ReadByte()
	_ = rd.UnreadByte()
	if char == '"' {
		fmt.Fscanf(rd, "%q ", &name)
		if len(name) == 0 {
			// TODO log.Error("Reader has no file name: reader=%+v", rd)
			return "", true
		}

		if name[0] == '\\' {
			name = name[1:]
		}
	} else {
		// This technique is potentially ambiguous it may not be possible to uniquely identify the filenames from the diff line alone
		ambiguity = true
		fmt.Fscanf(rd, "%s ", &name)
		char, _ := rd.ReadByte()
		_ = rd.UnreadByte()
		for !(char == 0 || char == '"' || char == 'b') {
			var suffix string
			fmt.Fscanf(rd, "%s ", &suffix)
			name += " " + suffix
			char, _ = rd.ReadByte()
			_ = rd.UnreadByte()
		}
	}
	if len(name) < 2 {
		// TODO log.Error("Unable to determine name from reader: reader=%+v", rd)
		return "", true
	}
	return name[2:], ambiguity
}

func createDiffFile(diff *Diff, line string) *DiffFile {
	// The a/ and b/ filenames are the same unless rename/copy is involved.
	// Especially, even for a creation or a deletion, /dev/null is not used
	// in place of the a/ or b/ filenames.
	//
	// When rename/copy is involved, file1 and file2 show the name of the
	// source file of the rename/copy and the name of the file that rename/copy
	// produces, respectively.
	//
	// Path names are quoted if necessary.
	//
	// This means that you should always be able to determine the file name even when there
	// there is potential ambiguity...
	//
	// but we can be simpler with our heuristics by just forcing git to prefix things nicely
	curFile := &DiffFile{
		Index:    len(diff.Files) + 1,
		Type:     DiffFileChange,
		Sections: make([]*DiffSection, 0, 10),
	}

	rd := strings.NewReader(line[len(cmdDiffHead):] + " ")
	curFile.Type = DiffFileChange
	var oldNameAmbiguity, newNameAmbiguity bool

	curFile.OldName, oldNameAmbiguity = readFileName(rd)
	curFile.Name, newNameAmbiguity = readFileName(rd)
	if oldNameAmbiguity && newNameAmbiguity {
		curFile.IsAmbiguous = true
		// OK we should bet that the oldName and the newName are the same if they can be made to be same
		// So we need to start again ...
		if (len(line)-len(cmdDiffHead)-1)%2 == 0 {
			// diff --git a/b b/b b/b b/b b/b b/b
			//
			midpoint := (len(line) + len(cmdDiffHead) - 1) / 2
			new, old := line[len(cmdDiffHead):midpoint], line[midpoint+1:]
			if len(new) > 2 && len(old) > 2 && new[2:] == old[2:] {
				curFile.OldName = old[2:]
				curFile.Name = old[2:]
			}
		}
	}

	curFile.IsRenamed = curFile.Name != curFile.OldName
	return curFile
}

func skipToNextDiffHead(input *bufio.Reader) (line string, err error) {
	// need to skip until the next cmdDiffHead
	var isFragment, wasFragment bool
	var lineBytes []byte
	for {
		lineBytes, isFragment, err = input.ReadLine()
		if err != nil {
			return
		}
		if wasFragment {
			wasFragment = isFragment
			continue
		}
		if bytes.HasPrefix(lineBytes, []byte(cmdDiffHead)) {
			break
		}
		wasFragment = isFragment
	}
	line = string(lineBytes)
	if isFragment {
		var tail string
		tail, err = input.ReadString('\n')
		if err != nil {
			return
		}
		line += tail
	}
	return line, err
}

// ParseDiffHunkString parse the diffhunk content and return
func ParseDiffHunkString(diffhunk string) (leftLine, leftHunk, rightLine, righHunk int) {
	ss := strings.Split(diffhunk, "@@")
	ranges := strings.Split(ss[1][1:], " ")
	leftRange := strings.Split(ranges[0], ",")
	leftLine, _ = strconv.Atoi(leftRange[0][1:])
	if len(leftRange) > 1 {
		leftHunk, _ = strconv.Atoi(leftRange[1])
	}
	if len(ranges) > 1 {
		rightRange := strings.Split(ranges[1], ",")
		rightLine, _ = strconv.Atoi(rightRange[0])
		if len(rightRange) > 1 {
			righHunk, _ = strconv.Atoi(rightRange[1])
		}
	} else {
		// TODO log.Debug("Parse line number failed: %v", diffhunk)
		rightLine = leftLine
		righHunk = leftHunk
	}
	return leftLine, leftHunk, rightLine, righHunk
}

func getDiffLineSectionInfo(treePath, line string, lastLeftIdx, lastRightIdx int) *DiffLineSectionInfo {
	leftLine, leftHunk, rightLine, righHunk := ParseDiffHunkString(line)

	return &DiffLineSectionInfo{
		Path:          treePath,
		LastLeftIdx:   lastLeftIdx,
		LastRightIdx:  lastRightIdx,
		LeftIdx:       leftLine,
		RightIdx:      rightLine,
		LeftHunkSize:  leftHunk,
		RightHunkSize: righHunk,
	}
}

const (
	blobSizeCutoff = 1024

	// MetaFileIdentifier is the string appearing at the first line of LFS pointer files.
	// https://github.com/git-lfs/git-lfs/blob/master/docs/spec.md
	MetaFileIdentifier = "version https://git-lfs.github.com/spec/v1"

	// MetaFileOidPrefix appears in LFS pointer files on a line before the sha256 hash.
	MetaFileOidPrefix = "oid sha256:"
)

// Pointer contains LFS pointer data
type Pointer struct {
	Oid  string `json:"oid" xorm:"UNIQUE(s) INDEX NOT NULL"`
	Size int64  `json:"size" xorm:"NOT NULL"`
}

// LFSMetaObject stores metadata for LFS tracked files.
type LFSMetaObject struct {
	ID           int64 `xorm:"pk autoincr"`
	Pointer      `xorm:"extends"`
	RepositoryID int64     `xorm:"UNIQUE(s) INDEX NOT NULL"`
	Existing     bool      `xorm:"-"`
	CreatedUnix  TimeStamp `xorm:"created"`
}

func parseHunks(curFile *DiffFile, maxLines, maxLineCharacters int, input *bufio.Reader) (lineBytes []byte, isFragment bool, err error) {
	sb := strings.Builder{}

	var (
		curSection        *DiffSection
		curFileLinesCount int
		curFileLFSPrefix  bool
	)

	lastLeftIdx := -1
	leftLine, rightLine := 1, 1

	for {
		for isFragment {
			curFile.IsIncomplete = true
			curFile.IsIncompleteLineTooLong = true
			_, isFragment, err = input.ReadLine()
			if err != nil {
				// Now by the definition of ReadLine this cannot be io.EOF
				err = fmt.Errorf("unable to ReadLine: %w", err)
				return
			}
		}
		sb.Reset()
		lineBytes, isFragment, err = input.ReadLine()
		if err != nil {
			if err == io.EOF {
				return
			}
			err = fmt.Errorf("unable to ReadLine: %w", err)
			return
		}
		if lineBytes[0] == 'd' {
			// End of hunks
			return
		}

		switch lineBytes[0] {
		case '@':
			if maxLines > -1 && curFileLinesCount >= maxLines {
				curFile.IsIncomplete = true
				continue
			}

			_, _ = sb.Write(lineBytes)
			for isFragment {
				// This is very odd indeed - we're in a section header and the line is too long
				// This really shouldn't happen...
				lineBytes, isFragment, err = input.ReadLine()
				if err != nil {
					// Now by the definition of ReadLine this cannot be io.EOF
					err = fmt.Errorf("unable to ReadLine: %w", err)
					return
				}
				_, _ = sb.Write(lineBytes)
			}
			line := sb.String()

			// Create a new section to represent this hunk
			curSection = &DiffSection{file: curFile}
			lastLeftIdx = -1
			curFile.Sections = append(curFile.Sections, curSection)

			lineSectionInfo := getDiffLineSectionInfo(curFile.Name, line, leftLine-1, rightLine-1)
			diffLine := &DiffLine{
				Type:        DiffLineSection,
				Content:     line,
				SectionInfo: lineSectionInfo,
			}
			curSection.Lines = append(curSection.Lines, diffLine)
			curSection.FileName = curFile.Name
			// update line number.
			leftLine = lineSectionInfo.LeftIdx
			rightLine = lineSectionInfo.RightIdx
			continue
		case '\\':
			if maxLines > -1 && curFileLinesCount >= maxLines {
				curFile.IsIncomplete = true
				continue
			}
			// This is used only to indicate that the current file does not have a terminal newline
			if !bytes.Equal(lineBytes, []byte("\\ No newline at end of file")) {
				err = fmt.Errorf("unexpected line in hunk: %s", string(lineBytes))
				return
			}
			// Technically this should be the end the file!
			// FIXME: we should be putting a marker at the end of the file if there is no terminal new line
			continue
		case '+':
			curFileLinesCount++
			curFile.Addition++
			if maxLines > -1 && curFileLinesCount >= maxLines {
				curFile.IsIncomplete = true
				continue
			}
			diffLine := &DiffLine{Type: DiffLineAdd, RightIdx: rightLine, Match: -1}
			rightLine++
			if curSection == nil {
				// Create a new section to represent this hunk
				curSection = &DiffSection{file: curFile}
				curFile.Sections = append(curFile.Sections, curSection)
				lastLeftIdx = -1
			}
			if lastLeftIdx > -1 {
				diffLine.Match = lastLeftIdx
				curSection.Lines[lastLeftIdx].Match = len(curSection.Lines)
				lastLeftIdx++
				if lastLeftIdx >= len(curSection.Lines) || curSection.Lines[lastLeftIdx].Type != DiffLineDel {
					lastLeftIdx = -1
				}
			}
			curSection.Lines = append(curSection.Lines, diffLine)
		case '-':
			curFileLinesCount++
			curFile.Deletion++
			if maxLines > -1 && curFileLinesCount >= maxLines {
				curFile.IsIncomplete = true
				continue
			}
			diffLine := &DiffLine{Type: DiffLineDel, LeftIdx: leftLine, Match: -1}
			if leftLine > 0 {
				leftLine++
			}
			if curSection == nil {
				// Create a new section to represent this hunk
				curSection = &DiffSection{file: curFile}
				curFile.Sections = append(curFile.Sections, curSection)
				lastLeftIdx = -1
			}
			if len(curSection.Lines) == 0 || curSection.Lines[len(curSection.Lines)-1].Type != DiffLineDel {
				lastLeftIdx = len(curSection.Lines)
			}
			curSection.Lines = append(curSection.Lines, diffLine)
		case ' ':
			curFileLinesCount++
			if maxLines > -1 && curFileLinesCount >= maxLines {
				curFile.IsIncomplete = true
				continue
			}
			diffLine := &DiffLine{Type: DiffLinePlain, LeftIdx: leftLine, RightIdx: rightLine}
			leftLine++
			rightLine++
			lastLeftIdx = -1
			if curSection == nil {
				// Create a new section to represent this hunk
				curSection = &DiffSection{file: curFile}
				curFile.Sections = append(curFile.Sections, curSection)
			}
			curSection.Lines = append(curSection.Lines, diffLine)
		default:
			// This is unexpected
			err = fmt.Errorf("unexpected line in hunk: %s", string(lineBytes))
			return
		}

		line := string(lineBytes)
		if isFragment {
			curFile.IsIncomplete = true
			curFile.IsIncompleteLineTooLong = true
			for isFragment {
				lineBytes, isFragment, err = input.ReadLine()
				if err != nil {
					// Now by the definition of ReadLine this cannot be io.EOF
					err = fmt.Errorf("unable to ReadLine: %w", err)
					return
				}
			}
		}
		if len(line) > maxLineCharacters {
			curFile.IsIncomplete = true
			curFile.IsIncompleteLineTooLong = true
			line = line[:maxLineCharacters]
		}
		curSection.Lines[len(curSection.Lines)-1].Content = line

		// handle LFS
		if line[1:] == MetaFileIdentifier {
			curFileLFSPrefix = true
		} else if curFileLFSPrefix && strings.HasPrefix(line[1:], MetaFileOidPrefix) {
			oid := strings.TrimPrefix(line[1:], MetaFileOidPrefix)
			if len(oid) == 64 {
				/* TODO put this back in if needed:
				m := &LFSMetaObject{Pointer: Pointer{Oid: oid}}
				count, err := db.CountByBean(db.DefaultContext, m)

				if err == nil && count > 0 {
					curFile.IsBin = true
					curFile.IsLFSFile = true
					curSection.Lines = nil
					lastLeftIdx = -1
				}*/
			}
		}
	}
}

// EncodeSha1 string to sha1 hex value.
func EncodeSha1(str string) string {
	h := sha1.New()
	_, _ = h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

// DetectEncoding detect the encoding of content
func DetectEncoding(content []byte) (string, error) {
	// First we check if the content represents valid utf8 content excepting a truncated character at the end.

	// Now we could decode all the runes in turn but this is not necessarily the cheapest thing to do
	// instead we walk backwards from the end to trim off a the incomplete character
	toValidate := content
	end := len(toValidate) - 1

	if end < 0 {
		// no-op
	} else if toValidate[end]>>5 == 0b110 {
		// Incomplete 1 byte extension e.g. © <c2><a9> which has been truncated to <c2>
		toValidate = toValidate[:end]
	} else if end > 0 && toValidate[end]>>6 == 0b10 && toValidate[end-1]>>4 == 0b1110 {
		// Incomplete 2 byte extension e.g. ⛔ <e2><9b><94> which has been truncated to <e2><9b>
		toValidate = toValidate[:end-1]
	} else if end > 1 && toValidate[end]>>6 == 0b10 && toValidate[end-1]>>6 == 0b10 && toValidate[end-2]>>3 == 0b11110 {
		// Incomplete 3 byte extension e.g. 💩 <f0><9f><92><a9> which has been truncated to <f0><9f><92>
		toValidate = toValidate[:end-2]
	}
	if utf8.Valid(toValidate) {
		// TODO log.Debug("Detected encoding: utf-8 (fast)")
		return "UTF-8", nil
	}

	textDetector := chardet.NewTextDetector()
	var detectContent []byte
	if len(content) < 1024 {
		// Check if original content is valid
		if _, err := textDetector.DetectBest(content); err != nil {
			return "", err
		}
		times := 1024 / len(content)
		detectContent = make([]byte, 0, times*len(content))
		for i := 0; i < times; i++ {
			detectContent = append(detectContent, content...)
		}
	} else {
		detectContent = content
	}

	// Now we can't use DetectBest or just results[0] because the result isn't stable - so we need a tie break
	results, err := textDetector.DetectAll(detectContent)
	if err != nil {
		if err == chardet.NotDetectedError && len(setting_Repository.AnsiCharset) > 0 {
			// TODO log.Debug("Using default AnsiCharset: %s", setting_Repository.AnsiCharset)
			return setting_Repository.AnsiCharset, nil
		}
		return "", err
	}

	topConfidence := results[0].Confidence
	topResult := results[0]
	priority, has := setting_Repository.DetectedCharsetScore[strings.ToLower(strings.TrimSpace(topResult.Charset))]
	for _, result := range results {
		// As results are sorted in confidence order - if we have a different confidence
		// we know it's less than the current confidence and can break out of the loop early
		if result.Confidence != topConfidence {
			break
		}

		// Otherwise check if this results is earlier in the DetectedCharsetOrder than our current top guess
		resultPriority, resultHas := setting_Repository.DetectedCharsetScore[strings.ToLower(strings.TrimSpace(result.Charset))]
		if resultHas && (!has || resultPriority < priority) {
			topResult = result
			priority = resultPriority
			has = true
		}
	}

	// FIXME: to properly decouple this function the fallback ANSI charset should be passed as an argument
	if topResult.Charset != "UTF-8" && len(setting_Repository.AnsiCharset) > 0 {
		// TODO log.Debug("Using default AnsiCharset: %s", setting.Repository.AnsiCharset)
		return setting_Repository.AnsiCharset, err
	}

	// TODO log.Debug("Detected encoding: %s", topResult.Charset)
	return topResult.Charset, err
}

const cmdDiffHead = "diff --git "

// ParsePatch builds a Diff object from a io.Reader and some parameters.
func ParsePatch(maxLines, maxLineCharacters, maxFiles int, reader io.Reader, skipToFile string) (*Diff, error) {
	// TODO log.Debug("ParsePatch(%d, %d, %d, ..., %s)", maxLines, maxLineCharacters, maxFiles, skipToFile)
	var curFile *DiffFile

	skipping := skipToFile != ""

	diff := &Diff{Files: make([]*DiffFile, 0)}

	sb := strings.Builder{}

	// OK let's set a reasonable buffer size.
	// This should be let's say at least the size of maxLineCharacters or 4096 whichever is larger.
	readerSize := maxLineCharacters
	if readerSize < 4096 {
		readerSize = 4096
	}

	input := bufio.NewReaderSize(reader, readerSize)
	line, err := input.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return diff, nil
		}
		return diff, err
	}
parsingLoop:
	for {
		// 1. A patch file always begins with `diff --git ` + `a/path b/path` (possibly quoted)
		// if it does not we have bad input!
		if !strings.HasPrefix(line, cmdDiffHead) {
			return diff, fmt.Errorf("invalid first file line: %s", line)
		}

		if maxFiles > -1 && len(diff.Files) >= maxFiles {
			lastFile := createDiffFile(diff, line)
			diff.End = lastFile.Name
			diff.IsIncomplete = true
			_, err := io.Copy(io.Discard, reader)
			if err != nil {
				// By the definition of io.Copy this never returns io.EOF
				return diff, fmt.Errorf("error during io.Copy: %w", err)
			}
			break parsingLoop
		}

		curFile = createDiffFile(diff, line)
		if skipping {
			if curFile.Name != skipToFile {
				line, err = skipToNextDiffHead(input)
				if err != nil {
					if err == io.EOF {
						return diff, nil
					}
					return diff, err
				}
				continue
			}
			skipping = false
		}

		diff.Files = append(diff.Files, curFile)

		// 2. It is followed by one or more extended header lines:
		//
		//     old mode <mode>
		//     new mode <mode>
		//     deleted file mode <mode>
		//     new file mode <mode>
		//     copy from <path>
		//     copy to <path>
		//     rename from <path>
		//     rename to <path>
		//     similarity index <number>
		//     dissimilarity index <number>
		//     index <hash>..<hash> <mode>
		//
		// * <mode> 6-digit octal numbers including the file type and file permission bits.
		// * <path> does not include the a/ and b/ prefixes
		// * <number> percentage of unchanged lines for similarity, percentage of changed
		//   lines dissimilarity as integer rounded down with terminal %. 100% => equal files.
		// * The index line includes the blob object names before and after the change.
		//   The <mode> is included if the file mode does not change; otherwise, separate
		//   lines indicate the old and the new mode.
		// 3. Following this header the "standard unified" diff format header may be encountered: (but not for every case...)
		//
		//     --- a/<path>
		//     +++ b/<path>
		//
		// With multiple hunks
		//
		//     @@ <hunk descriptor> @@
		//     +added line
		//     -removed line
		//      unchanged line
		//
		// 4. Binary files get:
		//
		//     Binary files a/<path> and b/<path> differ
		//
		// but one of a/<path> and b/<path> could be /dev/null.
	curFileLoop:
		for {
			line, err = input.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					return diff, err
				}
				break parsingLoop
			}
			switch {
			case strings.HasPrefix(line, cmdDiffHead):
				break curFileLoop
			case strings.HasPrefix(line, "old mode ") ||
				strings.HasPrefix(line, "new mode "):
				if strings.HasSuffix(line, " 160000\n") {
					curFile.IsSubmodule = true
				}
			case strings.HasPrefix(line, "rename from "):
				curFile.IsRenamed = true
				curFile.Type = DiffFileRename
				if curFile.IsAmbiguous {
					curFile.OldName = line[len("rename from ") : len(line)-1]
				}
			case strings.HasPrefix(line, "rename to "):
				curFile.IsRenamed = true
				curFile.Type = DiffFileRename
				if curFile.IsAmbiguous {
					curFile.Name = line[len("rename to ") : len(line)-1]
					curFile.IsAmbiguous = false
				}
			case strings.HasPrefix(line, "copy from "):
				curFile.IsRenamed = true
				curFile.Type = DiffFileCopy
				if curFile.IsAmbiguous {
					curFile.OldName = line[len("copy from ") : len(line)-1]
				}
			case strings.HasPrefix(line, "copy to "):
				curFile.IsRenamed = true
				curFile.Type = DiffFileCopy
				if curFile.IsAmbiguous {
					curFile.Name = line[len("copy to ") : len(line)-1]
					curFile.IsAmbiguous = false
				}
			case strings.HasPrefix(line, "new file"):
				curFile.Type = DiffFileAdd
				curFile.IsCreated = true
				if strings.HasSuffix(line, " 160000\n") {
					curFile.IsSubmodule = true
				}
			case strings.HasPrefix(line, "deleted"):
				curFile.Type = DiffFileDel
				curFile.IsDeleted = true
				if strings.HasSuffix(line, " 160000\n") {
					curFile.IsSubmodule = true
				}
			case strings.HasPrefix(line, "index"):
				if strings.HasSuffix(line, " 160000\n") {
					curFile.IsSubmodule = true
				}
			case strings.HasPrefix(line, "similarity index 100%"):
				curFile.Type = DiffFileRename
			case strings.HasPrefix(line, "Binary"):
				curFile.IsBin = true
			case strings.HasPrefix(line, "--- "):
				// Handle ambiguous filenames
				if curFile.IsAmbiguous {
					// The shortest string that can end up here is:
					// "--- a\t\n" without the quotes.
					// This line has a len() of 7 but doesn't contain a oldName.
					// So the amount that the line need is at least 8 or more.
					// The code will otherwise panic for a out-of-bounds.
					if len(line) > 7 && line[4] == 'a' {
						curFile.OldName = line[6 : len(line)-1]
						if line[len(line)-2] == '\t' {
							curFile.OldName = curFile.OldName[:len(curFile.OldName)-1]
						}
					} else {
						curFile.OldName = ""
					}
				}
				// Otherwise do nothing with this line
			case strings.HasPrefix(line, "+++ "):
				// Handle ambiguous filenames
				if curFile.IsAmbiguous {
					if len(line) > 6 && line[4] == 'b' {
						curFile.Name = line[6 : len(line)-1]
						if line[len(line)-2] == '\t' {
							curFile.Name = curFile.Name[:len(curFile.Name)-1]
						}
						if curFile.OldName == "" {
							curFile.OldName = curFile.Name
						}
					} else {
						curFile.Name = curFile.OldName
					}
					curFile.IsAmbiguous = false
				}
				// Otherwise do nothing with this line, but now switch to parsing hunks
				lineBytes, isFragment, err := parseHunks(curFile, maxLines, maxLineCharacters, input)
				diff.TotalAddition += curFile.Addition
				diff.TotalDeletion += curFile.Deletion
				if err != nil {
					if err != io.EOF {
						return diff, err
					}
					break parsingLoop
				}
				sb.Reset()
				_, _ = sb.Write(lineBytes)
				for isFragment {
					lineBytes, isFragment, err = input.ReadLine()
					if err != nil {
						// Now by the definition of ReadLine this cannot be io.EOF
						return diff, fmt.Errorf("unable to ReadLine: %w", err)
					}
					_, _ = sb.Write(lineBytes)
				}
				line = sb.String()
				sb.Reset()

				break curFileLoop
			}
		}
	}

	// TODO: There are numerous issues with this:
	// - we might want to consider detecting encoding while parsing but...
	// - we're likely to fail to get the correct encoding here anyway as we won't have enough information
	diffLineTypeBuffers := make(map[DiffLineType]*bytes.Buffer, 3)
	diffLineTypeDecoders := make(map[DiffLineType]*encoding.Decoder, 3)
	diffLineTypeBuffers[DiffLinePlain] = new(bytes.Buffer)
	diffLineTypeBuffers[DiffLineAdd] = new(bytes.Buffer)
	diffLineTypeBuffers[DiffLineDel] = new(bytes.Buffer)
	for _, f := range diff.Files {
		f.NameHash = EncodeSha1(f.Name)

		for _, buffer := range diffLineTypeBuffers {
			buffer.Reset()
		}
		for _, sec := range f.Sections {
			for _, l := range sec.Lines {
				if l.Type == DiffLineSection {
					continue
				}
				diffLineTypeBuffers[l.Type].WriteString(l.Content[1:])
				diffLineTypeBuffers[l.Type].WriteString("\n")
			}
		}
		for lineType, buffer := range diffLineTypeBuffers {
			diffLineTypeDecoders[lineType] = nil
			if buffer.Len() == 0 {
				continue
			}
			charsetLabel, err := DetectEncoding(buffer.Bytes())
			if charsetLabel != "UTF-8" && err == nil {
				encoding, _ := stdcharset.Lookup(charsetLabel)
				if encoding != nil {
					diffLineTypeDecoders[lineType] = encoding.NewDecoder()
				}
			}
		}
		for _, sec := range f.Sections {
			for _, l := range sec.Lines {
				decoder := diffLineTypeDecoders[l.Type]
				if decoder != nil {
					if c, _, err := transform.String(decoder, l.Content[1:]); err == nil {
						l.Content = l.Content[0:1] + c
					}
				}
			}
		}
	}

	diff.NumFiles = len(diff.Files)
	return diff, nil
}

type attributeTriple struct {
	Filename  string
	Attribute string
	Value     string
}

type attributeWriter interface {
	io.WriteCloser
	ReadAttribute() <-chan attributeTriple
}

// CheckAttributeReader provides a reader for check-attribute content that can be long running
type CheckAttributeReader struct {
	// params
	Attributes []CmdArg
	Repo       *Repository
	IndexFile  string
	WorkTree   string

	stdinReader io.ReadCloser
	stdinWriter *os.File
	stdOut      attributeWriter
	cmd         *Command
	env         []string
	ctx         context.Context
	cancel      context.CancelFunc
}

const windowsSharingViolationError syscall.Errno = 32

// RemoveAll removes the named file or (empty) directory with at most 5 attempts.
func RemoveAll(name string) error {
	var err error
	for i := 0; i < 5; i++ {
		err = os.RemoveAll(name)
		if err == nil {
			break
		}
		unwrapped := err.(*os.PathError).Err
		if unwrapped == syscall.EBUSY || unwrapped == syscall.ENOTEMPTY || unwrapped == syscall.EPERM || unwrapped == syscall.EMFILE || unwrapped == syscall.ENFILE {
			// try again
			<-time.After(100 * time.Millisecond)
			continue
		}

		if unwrapped == windowsSharingViolationError && runtime.GOOS == "windows" {
			// try again
			<-time.After(100 * time.Millisecond)
			continue
		}

		if unwrapped == syscall.ENOENT {
			// it's already gone
			return nil
		}
	}
	return err
}

// AddDynamicArguments adds new dynamic argument(s) to the command.
// The arguments may come from user input and can not be trusted, so no leading '-' is allowed to avoid passing options
func (c *Command) AddDynamicArguments(args ...string) *Command {
	for _, arg := range args {
		if arg != "" && arg[0] == '-' {
			c.brokenArgs = append(c.brokenArgs, arg)
		}
	}
	if len(c.brokenArgs) != 0 {
		return c
	}
	c.args = append(c.args, args...)
	return c
}

func (repo *Repository) readTreeToIndex(id SHA1, indexFilename ...string) error {
	var env []string
	if len(indexFilename) > 0 {
		env = append(os.Environ(), "GIT_INDEX_FILE="+indexFilename[0])
	}
	_, _, err := NewCommand(repo.Ctx, "read-tree").AddDynamicArguments(id.String()).RunStdString(&RunOpts{Dir: repo.Path, Env: env})
	if err != nil {
		return err
	}
	return nil
}

// ReadTreeToIndex reads a treeish to the index
func (repo *Repository) ReadTreeToIndex(treeish string, indexFilename ...string) error {
	if len(treeish) != 40 {
		res, _, err := NewCommand(repo.Ctx, "rev-parse", "--verify").AddDynamicArguments(treeish).RunStdString(&RunOpts{Dir: repo.Path})
		if err != nil {
			return err
		}
		if len(res) > 0 {
			treeish = res[:len(res)-1]
		}
	}
	id, err := NewIDFromString(treeish)
	if err != nil {
		return err
	}
	return repo.readTreeToIndex(id, indexFilename...)
}

// ReadTreeToTemporaryIndex reads a treeish to a temporary index file
func (repo *Repository) ReadTreeToTemporaryIndex(treeish string) (filename, tmpDir string, cancel context.CancelFunc, err error) {
	tmpDir, err = os.MkdirTemp("", "index")
	if err != nil {
		return
	}

	filename = filepath.Join(tmpDir, ".tmp-index")
	cancel = func() {
		err := RemoveAll(tmpDir)
		if err != nil {
			// TODO log.Error("failed to remove tmp index file: %v", err)
		}
	}
	err = repo.ReadTreeToIndex(treeish, filename)
	if err != nil {
		defer cancel()
		return "", "", func() {}, err
	}
	return filename, tmpDir, cancel, err
}

type nulSeparatedAttributeWriter struct {
	tmp        []byte
	attributes chan attributeTriple
	closed     chan struct{}
	working    attributeTriple
	pos        int
}

func (wr *nulSeparatedAttributeWriter) Write(p []byte) (n int, err error) {
	l, read := len(p), 0

	nulIdx := bytes.IndexByte(p, '\x00')
	for nulIdx >= 0 {
		wr.tmp = append(wr.tmp, p[:nulIdx]...)
		switch wr.pos {
		case 0:
			wr.working = attributeTriple{
				Filename: string(wr.tmp),
			}
		case 1:
			wr.working.Attribute = string(wr.tmp)
		case 2:
			wr.working.Value = string(wr.tmp)
		}
		wr.tmp = wr.tmp[:0]
		wr.pos++
		if wr.pos > 2 {
			wr.attributes <- wr.working
			wr.pos = 0
		}
		read += nulIdx + 1
		if l > read {
			p = p[nulIdx+1:]
			nulIdx = bytes.IndexByte(p, '\x00')
		} else {
			return l, nil
		}
	}
	wr.tmp = append(wr.tmp, p...)
	return len(p), nil
}

func (wr *nulSeparatedAttributeWriter) ReadAttribute() <-chan attributeTriple {
	return wr.attributes
}

func (wr *nulSeparatedAttributeWriter) Close() error {
	select {
	case <-wr.closed:
		return nil
	default:
	}
	close(wr.attributes)
	close(wr.closed)
	return nil
}

// Init initializes the CheckAttributeReader
func (c *CheckAttributeReader) Init(ctx context.Context) error {
	cmdArgs := []CmdArg{"check-attr", "--stdin", "-z"}

	if len(c.IndexFile) > 0 {
		cmdArgs = append(cmdArgs, "--cached")
		c.env = append(c.env, "GIT_INDEX_FILE="+c.IndexFile)
	}

	if len(c.WorkTree) > 0 {
		c.env = append(c.env, "GIT_WORK_TREE="+c.WorkTree)
	}

	c.env = append(c.env, "GIT_FLUSH=1")

	if len(c.Attributes) == 0 {
		lw := new(nulSeparatedAttributeWriter)
		lw.attributes = make(chan attributeTriple)
		lw.closed = make(chan struct{})

		c.stdOut = lw
		c.stdOut.Close()
		return fmt.Errorf("no provided Attributes to check")
	}

	cmdArgs = append(cmdArgs, c.Attributes...)
	cmdArgs = append(cmdArgs, "--")

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.cmd = NewCommand(c.ctx, cmdArgs...)

	var err error

	c.stdinReader, c.stdinWriter, err = os.Pipe()
	if err != nil {
		c.cancel()
		return err
	}

	lw := new(nulSeparatedAttributeWriter)
	lw.attributes = make(chan attributeTriple, 5)
	lw.closed = make(chan struct{})
	c.stdOut = lw
	return nil
}

// Run run cmd
func (c *CheckAttributeReader) Run() error {
	defer func() {
		_ = c.stdinReader.Close()
		_ = c.stdOut.Close()
	}()
	stdErr := new(bytes.Buffer)
	err := c.cmd.Run(&RunOpts{
		Env:    c.env,
		Dir:    c.Repo.Path,
		Stdin:  c.stdinReader,
		Stdout: c.stdOut,
		Stderr: stdErr,
	})
	if err != nil && //                      If there is an error we need to return but:
		c.ctx.Err() != err && //             1. Ignore the context error if the context is cancelled or exceeds the deadline (RunWithContext could return c.ctx.Err() which is Canceled or DeadlineExceeded)
		err.Error() != "signal: killed" { // 2. We should not pass up errors due to the program being killed
		return fmt.Errorf("failed to run attr-check. Error: %w\nStderr: %s", err, stdErr.String())
	}
	return nil
}

// CheckPath check attr for given path
func (c *CheckAttributeReader) CheckPath(path string) (rs map[string]string, err error) {
	defer func() {
		if err != nil && err != c.ctx.Err() {
			// TODO log.Error("Unexpected error when checking path %s in %s. Error: %v", path, c.Repo.Path, err)
		}
	}()

	select {
	case <-c.ctx.Done():
		return nil, c.ctx.Err()
	default:
	}

	if _, err = c.stdinWriter.Write([]byte(path + "\x00")); err != nil {
		defer c.Close()
		return nil, err
	}

	rs = make(map[string]string)
	for range c.Attributes {
		select {
		case attr, ok := <-c.stdOut.ReadAttribute():
			if !ok {
				return nil, c.ctx.Err()
			}
			rs[attr.Attribute] = attr.Value
		case <-c.ctx.Done():
			return nil, c.ctx.Err()
		}
	}
	return rs, nil
}

// Close close pip after use
func (c *CheckAttributeReader) Close() error {
	c.cancel()
	err := c.stdinWriter.Close()
	return err
}

// Create a check attribute reader for the current repository and provided commit ID
func (repo *Repository) CheckAttributeReader(commitID string) (*CheckAttributeReader, context.CancelFunc) {
	indexFilename, worktree, deleteTemporaryFile, err := repo.ReadTreeToTemporaryIndex(commitID)
	if err != nil {
		return nil, func() {}
	}

	checker := &CheckAttributeReader{
		Attributes: []CmdArg{"linguist-vendored", "linguist-generated", "linguist-language", "gitlab-language"},
		Repo:       repo,
		IndexFile:  indexFilename,
		WorkTree:   worktree,
	}
	ctx, cancel := context.WithCancel(repo.Ctx)
	if err := checker.Init(ctx); err != nil {
		// TODO log.Error("Unable to open checker for %s. Error: %v", commitID, err)
	} else {
		go func() {
			err := checker.Run()
			if err != nil && err != ctx.Err() {
				// TODO log.Error("Unable to open checker for %s. Error: %v", commitID, err)
			}
			cancel()
		}()
	}
	deferable := func() {
		_ = checker.Close()
		cancel()
		deleteTemporaryFile()
	}

	return checker, deferable
}

// IsVendor returns whether or not path is a vendor path.
func IsVendor(path string) bool {
	return enry.IsVendor(path)
}

// IsGenerated returns whether or not path is a generated path.
func IsGenerated(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if _, ok := data.GeneratedCodeExtensions[ext]; ok {
		return true
	}

	for _, m := range data.GeneratedCodeNameMatchers {
		if m(path) {
			return true
		}
	}

	return false
}

// Blob represents a Git object.
type Blob struct {
	ID SHA1

	gotSize bool
	size    int64
	name    string
	repo    *Repository
}

// ReadTreeID reads a tree ID from a cat-file --batch stream, throwing away the rest of the stream.
func ReadTreeID(rd *bufio.Reader, size int64) (string, error) {
	var id string
	var n int64
headerLoop:
	for {
		line, err := rd.ReadBytes('\n')
		if err != nil {
			return "", err
		}
		n += int64(len(line))
		idx := bytes.Index(line, []byte{' '})
		if idx < 0 {
			continue
		}

		if string(line[:idx]) == "tree" {
			id = string(line[idx+1 : len(line)-1])
			break headerLoop
		}
	}

	// Discard the rest of the commit
	discard := size - n + 1
	for discard > math.MaxInt32 {
		_, err := rd.Discard(math.MaxInt32)
		if err != nil {
			return id, err
		}
		discard -= math.MaxInt32
	}
	_, err := rd.Discard(int(discard))
	return id, err
}

// EntryMode the type of the object in the git tree
type EntryMode int

// There are only a few file modes in Git. They look like unix file modes, but they can only be
// one of these.
const (
	// EntryModeBlob
	EntryModeBlob EntryMode = 0o100644
	// EntryModeExec
	EntryModeExec EntryMode = 0o100755
	// EntryModeSymlink
	EntryModeSymlink EntryMode = 0o120000
	// EntryModeCommit
	EntryModeCommit EntryMode = 0o160000
	// EntryModeTree
	EntryModeTree EntryMode = 0o040000
)

// ParseTreeLine reads an entry from a tree in a cat-file --batch stream
// This carefully avoids allocations - except where fnameBuf is too small.
// It is recommended therefore to pass in an fnameBuf large enough to avoid almost all allocations
//
// Each line is composed of:
// <mode-in-ascii-dropping-initial-zeros> SP <fname> NUL <20-byte SHA>
//
// We don't attempt to convert the 20-byte SHA to 40-byte SHA to save a lot of time
func ParseTreeLine(rd *bufio.Reader, modeBuf, fnameBuf, shaBuf []byte) (mode, fname, sha []byte, n int, err error) {
	var readBytes []byte

	// Read the Mode & fname
	readBytes, err = rd.ReadSlice('\x00')
	if err != nil {
		return
	}
	idx := bytes.IndexByte(readBytes, ' ')
	if idx < 0 {
		// TODO log.Debug("missing space in readBytes ParseTreeLine: %s", readBytes)

		err = &ErrNotExist{}
		return
	}

	n += idx + 1
	copy(modeBuf, readBytes[:idx])
	if len(modeBuf) >= idx {
		modeBuf = modeBuf[:idx]
	} else {
		modeBuf = append(modeBuf, readBytes[len(modeBuf):idx]...)
	}
	mode = modeBuf

	readBytes = readBytes[idx+1:]

	// Deal with the fname
	copy(fnameBuf, readBytes)
	if len(fnameBuf) > len(readBytes) {
		fnameBuf = fnameBuf[:len(readBytes)]
	} else {
		fnameBuf = append(fnameBuf, readBytes[len(fnameBuf):]...)
	}
	for err == bufio.ErrBufferFull {
		readBytes, err = rd.ReadSlice('\x00')
		fnameBuf = append(fnameBuf, readBytes...)
	}
	n += len(fnameBuf)
	if err != nil {
		return
	}
	fnameBuf = fnameBuf[:len(fnameBuf)-1]
	fname = fnameBuf

	// Deal with the 20-byte SHA
	idx = 0
	for idx < 20 {
		var read int
		read, err = rd.Read(shaBuf[idx:20])
		n += read
		if err != nil {
			return
		}
		idx += read
	}
	sha = shaBuf
	return mode, fname, sha, n, err
}

// ParseTreeEntries parses the output of a `git ls-tree -l` command.
func ParseTreeEntries(data []byte) ([]*TreeEntry, error) {
	return parseTreeEntries(data, nil)
}

var sepSpace = []byte{' '}

func parseTreeEntries(data []byte, ptree *Tree) ([]*TreeEntry, error) {
	var err error
	entries := make([]*TreeEntry, 0, bytes.Count(data, []byte{'\n'})+1)
	for pos := 0; pos < len(data); {
		// expect line to be of the form:
		// <mode> <type> <sha> <space-padded-size>\t<filename>
		// <mode> <type> <sha>\t<filename>
		posEnd := bytes.IndexByte(data[pos:], '\n')
		if posEnd == -1 {
			posEnd = len(data)
		} else {
			posEnd += pos
		}
		line := data[pos:posEnd]
		posTab := bytes.IndexByte(line, '\t')
		if posTab == -1 {
			return nil, fmt.Errorf("invalid ls-tree output (no tab): %q", line)
		}

		entry := new(TreeEntry)
		entry.ptree = ptree

		entryAttrs := line[:posTab]
		entryName := line[posTab+1:]

		entryMode, entryAttrs, _ := bytes.Cut(entryAttrs, sepSpace)
		_ /* entryType */, entryAttrs, _ = bytes.Cut(entryAttrs, sepSpace) // the type is not used, the mode is enough to determine the type
		entryObjectID, entryAttrs, _ := bytes.Cut(entryAttrs, sepSpace)
		if len(entryAttrs) > 0 {
			entrySize := entryAttrs // the last field is the space-padded-size
			entry.size, _ = strconv.ParseInt(strings.TrimSpace(string(entrySize)), 10, 64)
			entry.sized = true
		}

		switch string(entryMode) {
		case "100644":
			entry.entryMode = EntryModeBlob
		case "100755":
			entry.entryMode = EntryModeExec
		case "120000":
			entry.entryMode = EntryModeSymlink
		case "160000":
			entry.entryMode = EntryModeCommit
		case "040000", "040755": // git uses 040000 for tree object, but some users may get 040755 for unknown reasons
			entry.entryMode = EntryModeTree
		default:
			return nil, fmt.Errorf("unknown type: %v", string(entryMode))
		}

		entry.ID, err = NewIDFromString(string(entryObjectID))
		if err != nil {
			return nil, fmt.Errorf("invalid ls-tree output (invalid object id): %q, err: %w", line, err)
		}

		if len(entryName) > 0 && entryName[0] == '"' {
			entry.name, err = strconv.Unquote(string(entryName))
			if err != nil {
				return nil, fmt.Errorf("invalid ls-tree output (invalid name): %q, err: %w", line, err)
			}
		} else {
			entry.name = string(entryName)
		}

		pos = posEnd + 1
		entries = append(entries, entry)
	}
	return entries, nil
}

func catBatchParseTreeEntries(ptree *Tree, rd *bufio.Reader, sz int64) ([]*TreeEntry, error) {
	fnameBuf := make([]byte, 4096)
	modeBuf := make([]byte, 40)
	shaBuf := make([]byte, 40)
	entries := make([]*TreeEntry, 0, 10)

loop:
	for sz > 0 {
		mode, fname, sha, count, err := ParseTreeLine(rd, modeBuf, fnameBuf, shaBuf)
		if err != nil {
			if err == io.EOF {
				break loop
			}
			return nil, err
		}
		sz -= int64(count)
		entry := new(TreeEntry)
		entry.ptree = ptree

		switch string(mode) {
		case "100644":
			entry.entryMode = EntryModeBlob
		case "100755":
			entry.entryMode = EntryModeExec
		case "120000":
			entry.entryMode = EntryModeSymlink
		case "160000":
			entry.entryMode = EntryModeCommit
		case "40000", "40755": // git uses 40000 for tree object, but some users may get 40755 for unknown reasons
			entry.entryMode = EntryModeTree
		default:
			// TODO log.Debug("Unknown mode: %v", string(mode))
			return nil, fmt.Errorf("unknown mode: %v", string(mode))
		}

		entry.ID = MustID(sha)
		entry.name = string(fname)
		entries = append(entries, entry)
	}
	if _, err := rd.Discard(1); err != nil {
		return entries, err
	}

	return entries, nil
}

// ListEntries returns all entries of current tree.
func (t *Tree) ListEntries() (Entries, error) {
	if t.entriesParsed {
		return t.entries, nil
	}

	if t.repo != nil {
		wr, rd, cancel := t.repo.CatFileBatch(t.repo.Ctx)
		defer cancel()

		_, _ = wr.Write([]byte(t.ID.String() + "\n"))
		_, typ, sz, err := ReadBatchLine(rd)
		if err != nil {
			return nil, err
		}
		if typ == "commit" {
			treeID, err := ReadTreeID(rd, sz)
			if err != nil && err != io.EOF {
				return nil, err
			}
			_, _ = wr.Write([]byte(treeID + "\n"))
			_, typ, sz, err = ReadBatchLine(rd)
			if err != nil {
				return nil, err
			}
		}
		if typ == "tree" {
			t.entries, err = catBatchParseTreeEntries(t, rd, sz)
			if err != nil {
				return nil, err
			}
			t.entriesParsed = true
			return t.entries, nil
		}

		// Not a tree just use ls-tree instead
		for sz > math.MaxInt32 {
			discarded, err := rd.Discard(math.MaxInt32)
			sz -= int64(discarded)
			if err != nil {
				return nil, err
			}
		}
		for sz > 0 {
			discarded, err := rd.Discard(int(sz))
			sz -= int64(discarded)
			if err != nil {
				return nil, err
			}
		}
	}

	stdout, _, runErr := NewCommand(t.repo.Ctx, "ls-tree", "-l").AddDynamicArguments(t.ID.String()).RunStdBytes(&RunOpts{Dir: t.repo.Path})
	if runErr != nil {
		if strings.Contains(runErr.Error(), "fatal: Not a valid object name") || strings.Contains(runErr.Error(), "fatal: not a tree object") {
			return nil, ErrNotExist{
				ID: t.ID.String(),
			}
		}
		return nil, runErr
	}

	var err error
	t.entries, err = parseTreeEntries(stdout, t)
	if err == nil {
		t.entriesParsed = true
	}

	return t.entries, err
}

func (repo *Repository) getTree(id SHA1) (*Tree, error) {
	wr, rd, cancel := repo.CatFileBatch(repo.Ctx)
	defer cancel()

	_, _ = wr.Write([]byte(id.String() + "\n"))

	// ignore the SHA
	_, typ, size, err := ReadBatchLine(rd)
	if err != nil {
		return nil, err
	}

	switch typ {
	case "tag":
		resolvedID := id
		data, err := io.ReadAll(io.LimitReader(rd, size))
		if err != nil {
			return nil, err
		}
		tag, err := parseTagData(data)
		if err != nil {
			return nil, err
		}
		commit, err := tag.Commit(repo)
		if err != nil {
			return nil, err
		}
		commit.Tree.ResolvedID = resolvedID
		return &commit.Tree, nil
	case "commit":
		commit, err := CommitFromReader(repo, id, io.LimitReader(rd, size))
		if err != nil {
			return nil, err
		}
		if _, err := rd.Discard(1); err != nil {
			return nil, err
		}
		commit.Tree.ResolvedID = commit.ID
		return &commit.Tree, nil
	case "tree":
		tree := NewTree(repo, id)
		tree.ResolvedID = id
		tree.entries, err = catBatchParseTreeEntries(tree, rd, size)
		if err != nil {
			return nil, err
		}
		tree.entriesParsed = true
		return tree, nil
	default:
		return nil, ErrNotExist{
			ID: id.String(),
		}
	}
}

// SubTree get a sub tree by the sub dir path
func (t *Tree) SubTree(rpath string) (*Tree, error) {
	if len(rpath) == 0 {
		return t, nil
	}

	paths := strings.Split(rpath, "/")
	var (
		err error
		g   = t
		p   = t
		te  *TreeEntry
	)
	for _, name := range paths {
		te, err = p.GetTreeEntryByPath(name)
		if err != nil {
			return nil, err
		}

		g, err = t.repo.getTree(te.ID)
		if err != nil {
			return nil, err
		}
		g.ptree = p
		p = g
	}
	return g, nil
}

// GetTreeEntryByPath get the tree entries according the sub dir
func (t *Tree) GetTreeEntryByPath(relpath string) (*TreeEntry, error) {
	if len(relpath) == 0 {
		return &TreeEntry{
			ptree:     t,
			ID:        t.ID,
			name:      "",
			fullName:  "",
			entryMode: EntryModeTree,
		}, nil
	}

	// FIXME: This should probably use git cat-file --batch to be a bit more efficient
	relpath = path.Clean(relpath)
	parts := strings.Split(relpath, "/")
	var err error
	tree := t
	for i, name := range parts {
		if i == len(parts)-1 {
			entries, err := tree.ListEntries()
			if err != nil {
				return nil, err
			}
			for _, v := range entries {
				if v.Name() == name {
					return v, nil
				}
			}
		} else {
			tree, err = tree.SubTree(name)
			if err != nil {
				return nil, err
			}
		}
	}
	return nil, ErrNotExist{"", relpath}
}

// IsDir if the entry is a sub dir
func (te *TreeEntry) IsDir() bool {
	return te.entryMode == EntryModeTree
}

// IsSubModule if the entry is a sub module
func (te *TreeEntry) IsSubModule() bool {
	return te.entryMode == EntryModeCommit
}

// Blob returns the blob object the entry
func (te *TreeEntry) Blob() *Blob {
	return &Blob{
		ID:      te.ID,
		name:    te.Name(),
		size:    te.size,
		gotSize: te.sized,
		repo:    te.ptree.repo,
	}
}

// GetBlobByPath get the blob object according the path
func (t *Tree) GetBlobByPath(relpath string) (*Blob, error) {
	entry, err := t.GetTreeEntryByPath(relpath)
	if err != nil {
		return nil, err
	}

	if !entry.IsDir() && !entry.IsSubModule() {
		return entry.Blob(), nil
	}

	return nil, ErrNotExist{"", relpath}
}

type blobReader struct {
	rd     *bufio.Reader
	n      int64
	cancel func()
}

func (b *blobReader) Read(p []byte) (n int, err error) {
	if b.n <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > b.n {
		p = p[0:b.n]
	}
	n, err = b.rd.Read(p)
	b.n -= int64(n)
	return n, err
}

// Close implements io.Closer
func (b *blobReader) Close() error {
	defer b.cancel()
	if b.n > 0 {
		for b.n > math.MaxInt32 {
			n, err := b.rd.Discard(math.MaxInt32)
			b.n -= int64(n)
			if err != nil {
				return err
			}
			b.n -= math.MaxInt32
		}
		n, err := b.rd.Discard(int(b.n))
		b.n -= int64(n)
		if err != nil {
			return err
		}
	}
	if b.n == 0 {
		_, err := b.rd.Discard(1)
		b.n--
		return err
	}
	return nil
}

// DataAsync gets a ReadCloser for the contents of a blob without reading it all.
// Calling the Close function on the result will discard all unread output.
func (b *Blob) DataAsync() (io.ReadCloser, error) {
	wr, rd, cancel := b.repo.CatFileBatch(b.repo.Ctx)

	_, err := wr.Write([]byte(b.ID.String() + "\n"))
	if err != nil {
		cancel()
		return nil, err
	}
	_, _, size, err := ReadBatchLine(rd)
	if err != nil {
		cancel()
		return nil, err
	}
	b.gotSize = true
	b.size = size

	if size < 4096 {
		bs, err := io.ReadAll(io.LimitReader(rd, size))
		defer cancel()
		if err != nil {
			return nil, err
		}
		_, err = rd.Discard(1)
		return io.NopCloser(bytes.NewReader(bs)), err
	}

	return &blobReader{
		rd:     rd,
		n:      size,
		cancel: cancel,
	}, nil
}

// GetBlobLineCount gets line count of the blob
func (b *Blob) GetBlobLineCount() (int, error) {
	reader, err := b.DataAsync()
	if err != nil {
		return 0, err
	}
	defer reader.Close()
	buf := make([]byte, 32*1024)
	count := 1
	lineSep := []byte{'\n'}

	c, err := reader.Read(buf)
	if c == 0 && err == io.EOF {
		return 0, nil
	}
	for {
		count += bytes.Count(buf[:c], lineSep)
		switch {
		case err == io.EOF:
			return count, nil
		case err != nil:
			return count, err
		}
		c, err = reader.Read(buf)
	}
}

func getCommitFileLineCount(commit *Commit, filePath string) int {
	blob, err := commit.GetBlobByPath(filePath)
	if err != nil {
		return 0
	}
	lineCount, err := blob.GetBlobLineCount()
	if err != nil {
		return 0
	}
	return lineCount
}

// GetTailSection creates a fake DiffLineSection if the last section is not the end of the file
func (diffFile *DiffFile) GetTailSection(gitRepo *Repository, leftCommitID, rightCommitID string) *DiffSection {
	if len(diffFile.Sections) == 0 || diffFile.Type != DiffFileChange || diffFile.IsBin || diffFile.IsLFSFile {
		return nil
	}
	leftCommit, err := gitRepo.GetCommit(leftCommitID)
	if err != nil {
		return nil
	}
	rightCommit, err := gitRepo.GetCommit(rightCommitID)
	if err != nil {
		return nil
	}
	lastSection := diffFile.Sections[len(diffFile.Sections)-1]
	lastLine := lastSection.Lines[len(lastSection.Lines)-1]
	leftLineCount := getCommitFileLineCount(leftCommit, diffFile.Name)
	rightLineCount := getCommitFileLineCount(rightCommit, diffFile.Name)
	if leftLineCount <= lastLine.LeftIdx || rightLineCount <= lastLine.RightIdx {
		return nil
	}
	tailDiffLine := &DiffLine{
		Type:    DiffLineSection,
		Content: " ",
		SectionInfo: &DiffLineSectionInfo{
			Path:         diffFile.Name,
			LastLeftIdx:  lastLine.LeftIdx,
			LastRightIdx: lastLine.RightIdx,
			LeftIdx:      leftLineCount,
			RightIdx:     rightLineCount,
		},
	}
	tailSection := &DiffSection{FileName: diffFile.Name, Lines: []*DiffLine{tailDiffLine}}
	return tailSection
}

var shortStatFormat = regexp.MustCompile(
	`\s*(\d+) files? changed(?:, (\d+) insertions?\(\+\))?(?:, (\d+) deletions?\(-\))?`)

var patchCommits = regexp.MustCompile(`^From\s(\w+)\s`)

func parseDiffStat(stdout string) (numFiles, totalAdditions, totalDeletions int, err error) {
	if len(stdout) == 0 || stdout == "\n" {
		return 0, 0, 0, nil
	}
	groups := shortStatFormat.FindStringSubmatch(stdout)
	if len(groups) != 4 {
		return 0, 0, 0, fmt.Errorf("unable to parse shortstat: %s groups: %s", stdout, groups)
	}

	numFiles, err = strconv.Atoi(groups[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("unable to parse shortstat: %s. Error parsing NumFiles %w", stdout, err)
	}

	if len(groups[2]) != 0 {
		totalAdditions, err = strconv.Atoi(groups[2])
		if err != nil {
			return 0, 0, 0, fmt.Errorf("unable to parse shortstat: %s. Error parsing NumAdditions %w", stdout, err)
		}
	}

	if len(groups[3]) != 0 {
		totalDeletions, err = strconv.Atoi(groups[3])
		if err != nil {
			return 0, 0, 0, fmt.Errorf("unable to parse shortstat: %s. Error parsing NumDeletions %w", stdout, err)
		}
	}
	return numFiles, totalAdditions, totalDeletions, err
}

// GetDiffShortStat counts number of changed files, number of additions and deletions
func GetDiffShortStat(ctx context.Context, repoPath string, args ...CmdArg) (numFiles, totalAdditions, totalDeletions int, err error) {
	// Now if we call:
	// $ git diff --shortstat 1ebb35b98889ff77299f24d82da426b434b0cca0...788b8b1440462d477f45b0088875
	// we get:
	// " 9902 files changed, 2034198 insertions(+), 298800 deletions(-)\n"
	args = append([]CmdArg{
		"diff",
		"--shortstat",
	}, args...)

	stdout, _, err := NewCommand(ctx, args...).RunStdString(&RunOpts{Dir: repoPath})
	if err != nil {
		return 0, 0, 0, err
	}

	return parseDiffStat(stdout)
}

// GetDiff builds a Diff between two commits of a repository.
// Passing the empty string as beforeCommitID returns a diff from the parent commit.
// The whitespaceBehavior is either an empty string or a git flag
func GetDiff(gitRepo *Repository, opts *DiffOptions, files ...string) (*Diff, error) {
	repoPath := gitRepo.Path

	commit, err := gitRepo.GetCommit(opts.AfterCommitID)
	if err != nil {
		return nil, err
	}

	argsLength := 6
	if len(opts.WhitespaceBehavior) > 0 {
		argsLength++
	}
	if len(opts.SkipTo) > 0 {
		argsLength++
	}
	if len(files) > 0 {
		argsLength += len(files) + 1
	}

	diffArgs := make([]CmdArg, 0, argsLength)
	if (len(opts.BeforeCommitID) == 0 || opts.BeforeCommitID == EmptySHA) && commit.ParentCount() == 0 {
		diffArgs = append(diffArgs, "diff", "--src-prefix=\\a/", "--dst-prefix=\\b/", "-M")
		if len(opts.WhitespaceBehavior) != 0 {
			diffArgs = append(diffArgs, opts.WhitespaceBehavior)
		}
		// append empty tree ref
		diffArgs = append(diffArgs, "4b825dc642cb6eb9a060e54bf8d69288fbee4904")
		diffArgs = append(diffArgs, CmdArgCheck(opts.AfterCommitID))
	} else {
		actualBeforeCommitID := opts.BeforeCommitID
		if len(actualBeforeCommitID) == 0 {
			parentCommit, _ := commit.Parent(0)
			actualBeforeCommitID = parentCommit.ID.String()
		}
		diffArgs = append(diffArgs, "diff", "--src-prefix=\\a/", "--dst-prefix=\\b/", "-M")
		if len(opts.WhitespaceBehavior) != 0 {
			diffArgs = append(diffArgs, opts.WhitespaceBehavior)
		}
		diffArgs = append(diffArgs, CmdArgCheck(actualBeforeCommitID))
		diffArgs = append(diffArgs, CmdArgCheck(opts.AfterCommitID))
		opts.BeforeCommitID = actualBeforeCommitID
	}

	// In git 2.31, git diff learned --skip-to which we can use to shortcut skip to file
	// so if we are using at least this version of git we don't have to tell ParsePatch to do
	// the skipping for us
	parsePatchSkipToFile := opts.SkipTo
	if opts.SkipTo != "" && CheckGitVersionAtLeast("2.31") == nil {
		diffArgs = append(diffArgs, CmdArg("--skip-to="+opts.SkipTo))
		parsePatchSkipToFile = ""
	}

	if len(files) > 0 {
		diffArgs = append(diffArgs, "--")
		for _, file := range files {
			diffArgs = append(diffArgs, CmdArg(file)) // it's safe to cast it to CmdArg because there is a "--" before
		}
	}

	reader, writer := io.Pipe()
	defer func() {
		_ = reader.Close()
		_ = writer.Close()
	}()

	go func(ctx context.Context, diffArgs []CmdArg, repoPath string, writer *io.PipeWriter) {
		cmd := NewCommand(ctx, diffArgs...)
		cmd.SetDescription(fmt.Sprintf("GetDiffRange [repo_path: %s]", repoPath))
		if err := cmd.Run(&RunOpts{
			Timeout: time.Duration(60 /* TODO setting.Git.Timeout.Default*/) * time.Second,
			Dir:     repoPath,
			Stderr:  os.Stderr,
			Stdout:  writer,
		}); err != nil {
			// TODO. log.Error("error during RunWithContext: %w", err)
		}

		_ = writer.Close()
	}(gitRepo.Ctx, diffArgs, repoPath, writer)

	diff, err := ParsePatch(opts.MaxLines, opts.MaxLineCharacters, opts.MaxFiles, reader, parsePatchSkipToFile)
	if err != nil {
		return nil, fmt.Errorf("unable to ParsePatch: %w", err)
	}
	diff.Start = opts.SkipTo

	checker, deferable := gitRepo.CheckAttributeReader(opts.AfterCommitID)
	defer deferable()

	for _, diffFile := range diff.Files {

		gotVendor := false
		gotGenerated := false
		if checker != nil {
			attrs, err := checker.CheckPath(diffFile.Name)
			if err == nil {
				if vendored, has := attrs["linguist-vendored"]; has {
					if vendored == "set" || vendored == "true" {
						diffFile.IsVendored = true
						gotVendor = true
					} else {
						gotVendor = vendored == "false"
					}
				}
				if generated, has := attrs["linguist-generated"]; has {
					if generated == "set" || generated == "true" {
						diffFile.IsGenerated = true
						gotGenerated = true
					} else {
						gotGenerated = generated == "false"
					}
				}
				if language, has := attrs["linguist-language"]; has && language != "unspecified" && language != "" {
					diffFile.Language = language
				} else if language, has := attrs["gitlab-language"]; has && language != "unspecified" && language != "" {
					diffFile.Language = language
				}
			}
		}

		if !gotVendor {
			diffFile.IsVendored = IsVendor(diffFile.Name)
		}
		if !gotGenerated {
			diffFile.IsGenerated = IsGenerated(diffFile.Name)
		}

		tailSection := diffFile.GetTailSection(gitRepo, opts.BeforeCommitID, opts.AfterCommitID)
		if tailSection != nil {
			diffFile.Sections = append(diffFile.Sections, tailSection)
		}
	}

	separator := "..."
	if opts.DirectComparison {
		separator = ".."
	}

	shortstatArgs := []CmdArg{CmdArgCheck(opts.BeforeCommitID + separator + opts.AfterCommitID)}
	if len(opts.BeforeCommitID) == 0 || opts.BeforeCommitID == EmptySHA {
		shortstatArgs = []CmdArg{EmptyTreeSHA, CmdArgCheck(opts.AfterCommitID)}
	}
	diff.NumFiles, diff.TotalAddition, diff.TotalDeletion, err = GetDiffShortStat(gitRepo.Ctx, repoPath, shortstatArgs...)
	if err != nil && strings.Contains(err.Error(), "no merge base") {
		// git >= 2.28 now returns an error if base and head have become unrelated.
		// previously it would return the results of git diff --shortstat base head so let's try that...
		shortstatArgs = []CmdArg{CmdArgCheck(opts.BeforeCommitID), CmdArgCheck(opts.AfterCommitID)}
		diff.NumFiles, diff.TotalAddition, diff.TotalDeletion, err = GetDiffShortStat(gitRepo.Ctx, repoPath, shortstatArgs...)
	}
	if err != nil {
		return nil, err
	}

	return diff, nil
}

/*
// UserPath returns the path absolute path of user repositories.
func UserPath(userName string) string { //revive:disable-line:exported
	return filepath.Join(RepoRootPath, strings.ToLower(userName))
}

// RepoPath returns repository path by given user and repository name.
func RepoPath(userName, repoName string) string { //revive:disable-line:exported
	return filepath.Join(UserPath(userName), strings.ToLower(repoName)+".git")
}

// RepoPath returns the repository path
func (repo *repo_Repository) RepoPath() string {
	return RepoPath(repo.OwnerName, repo.Name)
}

// CommitFileStatus represents status of files in a commit.
type CommitFileStatus struct {
	Added    []string
	Removed  []string
	Modified []string
}

// NewCommitFileStatus creates a CommitFileStatus
func NewCommitFileStatus() *CommitFileStatus {
	return &CommitFileStatus{
		[]string{}, []string{}, []string{},
	}
}

func parseCommitFileStatus(fileStatus *CommitFileStatus, stdout io.Reader) {
	rd := bufio.NewReader(stdout)
	peek, err := rd.Peek(1)
	if err != nil {
		if err != io.EOF {
			// TODO log.Error("Unexpected error whilst reading from git log --name-status. Error: %v", err)
		}
		return
	}
	if peek[0] == '\n' || peek[0] == '\x00' {
		_, _ = rd.Discard(1)
	}
	for {
		modifier, err := rd.ReadSlice('\x00')
		if err != nil {
			if err != io.EOF {
				// TODO log.Error("Unexpected error whilst reading from git log --name-status. Error: %v", err)
			}
			return
		}
		file, err := rd.ReadString('\x00')
		if err != nil {
			if err != io.EOF {
				// TODO log.Error("Unexpected error whilst reading from git log --name-status. Error: %v", err)
			}
			return
		}
		file = file[:len(file)-1]
		switch modifier[0] {
		case 'A':
			fileStatus.Added = append(fileStatus.Added, file)
		case 'D':
			fileStatus.Removed = append(fileStatus.Removed, file)
		case 'M':
			fileStatus.Modified = append(fileStatus.Modified, file)
		}
	}
}

// GetCommitFileStatus returns file status of commit in given repository.
func GetCommitFileStatus(ctx context.Context, repoPath, commitID string) (*CommitFileStatus, error) {
	stdout, w := io.Pipe()
	done := make(chan struct{})
	fileStatus := NewCommitFileStatus()
	go func() {
		parseCommitFileStatus(fileStatus, stdout)
		close(done)
	}()

	stderr := new(bytes.Buffer)
	err := NewCommand(ctx, "log", "--name-status", "-c", "--pretty=format:", "--parents", "--no-renames", "-z", "-1").AddDynamicArguments(commitID).Run(&RunOpts{
		Dir:    repoPath,
		Stdout: w,
		Stderr: stderr,
	})
	w.Close() // Close writer to exit parsing goroutine
	if err != nil {
		return nil, ConcatenateError(err, stderr.String())
	}

	<-done
	return fileStatus, nil
}

// CommitAffectedFiles store information about files affected by the commit
type CommitAffectedFiles struct {
	Filename string `json:"filename"`
}
*/
