package winston

type ItemMetadata struct {
	TMDBID               int    `json:"tmdb_id"`
	TVDBID               int    `json:"tvdb_id"`
	IMDBID               string `json:"imdb_id"`
	Kind                 string `json:"kind"`
	Title                string `json:"title"`
	Year                 int    `json:"year"`
	Season               int    `json:"season"`
	Episode              int    `json:"episode"`
	Quality              string `json:"quality"`
	ResolvedEpisodeTitle string `json:"resolved_episode_title"`
	ResolvedRelativePath  string `json:"resolved_relative_path"`
	RelativePathOverride string `json:"relative_path_override"`
}
