package settings

type Settings struct {
	ID              int64  `json:"id"`
	ApplicationName string `json:"application_name"`
	Timezone        string `json:"timezone"`
	WeekStartsOn    int    `json:"week_starts_on"`
	DatabasePath    string `json:"database_path"`
	BackupPath      string `json:"backup_path"`
	Theme           string `json:"theme"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}
