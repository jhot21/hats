package syncthing

type VersionInfo struct {
	Arch        string `json:"arch"`
	LongVersion string `json:"longVersion"`
	OS          string `json:"os"`
	Version     string `json:"version"`
}

type DeviceStatistics struct {
	LastSeen               string `json:"lastSeen"`
	LastConnectionDuration int    `json:"lastConnectionDurationS"`
}

type FolderConfig struct {
	ID                      string `json:"id"`
	Label                   string `json:"label,omitempty"`
	FilesystemType          string `json:"filesystemType,omitempty"`
	Path                    string `json:"path,omitempty"`
	Type                    string `json:"type,omitempty"`
	RescanIntervalS         int    `json:"rescanIntervalS,omitempty"`
	FsWatcherEnabled        bool   `json:"fsWatcherEnabled,omitempty"`
	FsWatcherDelayS         int    `json:"fsWatcherDelayS,omitempty"`
	IgnorePerms             bool   `json:"ignorePerms,omitempty"`
	AutoNormalize           bool   `json:"autoNormalize,omitempty"`
	Copiers                 int    `json:"copiers,omitempty"`
	PullerMaxPendingKiB     int    `json:"pullerMaxPendingKiB,omitempty"`
	Hashers                 int    `json:"hashers,omitempty"`
	Order                   string `json:"order,omitempty"`
	IgnoreDelete            bool   `json:"ignoreDelete,omitempty"`
	ScanProgressIntervalS   int    `json:"scanProgressIntervalS,omitempty"`
	PullerPauseS            int    `json:"pullerPauseS,omitempty"`
	MaxConflicts            int    `json:"maxConflicts,omitempty"`
	DisableSparseFiles      bool   `json:"disableSparseFiles,omitempty"`
	DisableTempIndexes      bool   `json:"disableTempIndexes,omitempty"`
	Paused                  bool   `json:"paused,omitempty"`
	WeakHashThresholdPct    int    `json:"weakHashThresholdPct,omitempty"`
	MarkerName              string `json:"markerName,omitempty"`
	CopyOwnershipFromParent bool   `json:"copyOwnershipFromParent,omitempty"`
	ModTimeWindowS          int    `json:"modTimeWindowS,omitempty"`
	MaxConcurrentWrites     int    `json:"maxConcurrentWrites,omitempty"`
	DisableFsync            bool   `json:"disableFsync,omitempty"`
	BlockPullOrder          string `json:"blockPullOrder,omitempty"`
	CopyRangeMethod         string `json:"copyRangeMethod,omitempty"`
	CaseSensitiveFS         bool   `json:"caseSensitiveFS,omitempty"`
	JunctionsAsDirs         bool   `json:"junctionsAsDirs,omitempty"`
	SyncOwnership           bool   `json:"syncOwnership,omitempty"`
	SendOwnership           bool   `json:"sendOwnership,omitempty"`
	SyncXattrs              bool   `json:"syncXattrs,omitempty"`
	SendXattrs              bool   `json:"sendXattrs,omitempty"`
}
