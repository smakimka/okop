package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"okop/pkg/sq"
	"okop/pkg/steam_proxy"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	pool *pgxpool.Pool

	developersCache *cache[string, int]
	publishersCache *cache[string, int]
}

func New(pool *pgxpool.Pool) *Repo {
	return &Repo{
		pool: pool,

		developersCache: newCache[string, int](),
		publishersCache: newCache[string, int](),
	}
}

func (r *Repo) GetAllGameSteamIDs(ctx context.Context) ([]int, error) {
	query, args, _ := sq.NewBuilder().Select("steam_id").From("apps_list").ToSql()
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	res, err := pgx.CollectRows(rows, pgx.RowTo[int])
	if err != nil {
		return nil, err
	}

	return res, nil
}

type pgxConn interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func (r *Repo) UpsertGame(ctx context.Context, appInfo *steam_proxy.StoreItem) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) // no-op after Commit

	developersIDs, err := r.getDevelopersIDs(ctx, tx, developerNames(appInfo))
	if err != nil {
		return err
	}
	publishersIDs, err := r.getPublishersIDs(ctx, tx, publisherNames(appInfo))
	if err != nil {
		return err
	}
	tagIDs, tagWeights := tagsFromAppInfo(appInfo)
	if err := r.ensureTagsExist(ctx, tx, tagIDs); err != nil {
		return err
	}

	game := gameFromAppInfo(appInfo)
	gameID, err := r.upsertGameRow(ctx, tx, game)
	if err != nil {
		return err
	}

	if err := r.replaceGameDevelopers(ctx, tx, gameID, developersIDs); err != nil {
		return err
	}
	if err := r.replaceGamePublishers(ctx, tx, gameID, publishersIDs); err != nil {
		return err
	}
	if err := r.replaceGameTags(ctx, tx, gameID, tagIDs, tagWeights); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func gameFromAppInfo(a *steam_proxy.StoreItem) *Game {
	g := &Game{
		SteamID:      int(a.ID),
		Name:         a.Name,
		StoreURLPath: a.StoreURLPath,
		IsFree:       a.IsFree,
	}
	if a.Platforms != nil {
		g.Windows = a.Platforms.Windows
		g.Mac = a.Platforms.Mac
		g.SteamOSLinux = a.Platforms.Linux
	}
	if a.Reviews != nil {
		if a.Reviews.Summary != nil {
			g.ReviewsFilteredCount = int(a.Reviews.Summary.ReviewCount)
			g.ReviewsFilteredPercentPositive = int(a.Reviews.Summary.PercentPositive)
			g.ReviewsFilteredScoreLabel = a.Reviews.Summary.ReviewScoreLabel
		}
		if a.Reviews.SummaryUnfiltered != nil {
			g.ReviewsUnfilteredCount = int(a.Reviews.SummaryUnfiltered.ReviewCount)
			g.ReviewsUnfilteredPercentPositive = int(a.Reviews.SummaryUnfiltered.PercentPositive)
			g.ReviewsUnfilteredScoreLabel = a.Reviews.SummaryUnfiltered.ReviewScoreLabel
		}
	}
	if a.Assets != nil {
		if raw, err := json.Marshal(a.Assets); err == nil {
			g.Assets = raw
		}
	}
	if a.Release != nil {
		g.ReleaseSteamDate = time.Unix(a.Release.SteamReleaseDate, 0).UTC()
		g.ReleaseOriginalSteamDate = time.Unix(a.Release.OriginalReleaseDate, 0).UTC()
	}
	return g
}

func developerNames(a *steam_proxy.StoreItem) []string {
	if a.BasicInfo == nil {
		return nil
	}
	out := make([]string, len(a.BasicInfo.Developers))
	for i, d := range a.BasicInfo.Developers {
		out[i] = d.Name
	}
	return out
}

func publisherNames(a *steam_proxy.StoreItem) []string {
	if a.BasicInfo == nil {
		return nil
	}
	out := make([]string, len(a.BasicInfo.Publishers))
	for i, p := range a.BasicInfo.Publishers {
		out[i] = p.Name
	}
	return out
}

func tagsFromAppInfo(a *steam_proxy.StoreItem) (ids []int, weights []int) {
	ids = make([]int, len(a.Tags))
	weights = make([]int, len(a.Tags))
	for i, t := range a.Tags {
		ids[i] = int(t.TagID)
		weights[i] = int(t.Weight)
	}
	return ids, weights
}

func (r *Repo) upsertGameRow(ctx context.Context, conn pgxConn, game *Game) (int, error) {
	query, args, _ := sq.NewBuilder().
		Insert("games").
		SetMap(map[string]any{
			"steam_id":                            game.SteamID,
			"name":                                game.Name,
			"store_url_path":                      game.StoreURLPath,
			"is_free":                             game.IsFree,
			"windows":                             game.Windows,
			"steamos_linux":                       game.SteamOSLinux,
			"mac":                                 game.Mac,
			"reviews_filtered_count":              game.ReviewsFilteredCount,
			"reviews_filtered_percent_positive":   game.ReviewsFilteredPercentPositive,
			"reviews_filtered_score_label":        game.ReviewsFilteredScoreLabel,
			"reviews_unfiltered_count":            game.ReviewsUnfilteredCount,
			"reviews_unfiltered_percent_positive": game.ReviewsUnfilteredPercentPositive,
			"reviews_unfiltered_score_label":      game.ReviewsUnfilteredScoreLabel,
			"assets":                              game.Assets,
			"release_steam_date":                  game.ReleaseSteamDate,
			"release_original_steam_date":         game.ReleaseOriginalSteamDate,
		}).
		Suffix(`on conflict (steam_id) do update set
			name = excluded.name,
			store_url_path = excluded.store_url_path,
			is_free = excluded.is_free,
			windows = excluded.windows,
			steamos_linux = excluded.steamos_linux,
			mac = excluded.mac,
			reviews_filtered_count = excluded.reviews_filtered_count,
			reviews_filtered_percent_positive = excluded.reviews_filtered_percent_positive,
			reviews_filtered_score_label = excluded.reviews_filtered_score_label,
			reviews_unfiltered_count = excluded.reviews_unfiltered_count,
			reviews_unfiltered_percent_positive = excluded.reviews_unfiltered_percent_positive,
			reviews_unfiltered_score_label = excluded.reviews_unfiltered_score_label,
			assets = excluded.assets,
			release_steam_date = excluded.release_steam_date,
			release_original_steam_date = excluded.release_original_steam_date
			returning id`).
		ToSql()
	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return pgx.CollectOneRow(rows, pgx.RowTo[int])
}

func (r *Repo) ensureTagsExist(ctx context.Context, conn pgxConn, tagIDs []int) error {
	for _, id := range tagIDs {
		query, args, _ := sq.NewBuilder().
			Insert("tags").
			Columns("id").
			Values(id).
			Suffix("on conflict (id) do nothing").
			ToSql()

		if _, err := conn.Exec(ctx, query, args...); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repo) replaceGameDevelopers(ctx context.Context, conn pgxConn, gameID int, developerIDs []int) error {
	if _, err := conn.Exec(ctx, `delete from game_developers where game_id = $1`, gameID); err != nil {
		return err
	}
	if len(developerIDs) == 0 {
		return nil
	}
	b := sq.NewBuilder().Insert("game_developers").Columns("game_id", "developer_id")
	for _, id := range developerIDs {
		b = b.Values(gameID, id)
	}
	query, args, _ := b.Suffix("on conflict do nothing").ToSql()
	_, err := conn.Exec(ctx, query, args...)
	return err
}

func (r *Repo) replaceGamePublishers(ctx context.Context, conn pgxConn, gameID int, publisherIDs []int) error {
	if _, err := conn.Exec(ctx, `delete from game_publishers where game_id = $1`, gameID); err != nil {
		return err
	}
	if len(publisherIDs) == 0 {
		return nil
	}
	b := sq.NewBuilder().Insert("game_publishers").Columns("game_id", "publisher_id")
	for _, id := range publisherIDs {
		b = b.Values(gameID, id)
	}
	query, args, _ := b.Suffix("on conflict do nothing").ToSql()
	_, err := conn.Exec(ctx, query, args...)
	return err
}

func (r *Repo) replaceGameTags(ctx context.Context, conn pgxConn, gameID int, tagIDs, weights []int) error {
	if _, err := conn.Exec(ctx, `delete from game_tags where game_id = $1`, gameID); err != nil {
		return err
	}
	if len(tagIDs) == 0 {
		return nil
	}
	b := sq.NewBuilder().Insert("game_tags").Columns("game_id", "tag_id", "weight")
	for i, tagID := range tagIDs {
		b = b.Values(gameID, tagID, weights[i])
	}
	query, args, _ := b.Suffix("on conflict do nothing").ToSql()
	if _, err := conn.Exec(ctx, query, args...); err != nil {
		return err
	}

	query, args, _ = sq.NewBuilder().
		Update("games").
		Set("tag_ids", tagIDs).
		Where(squirrel.Eq{"id": gameID}).
		ToSql()

	if _, err := conn.Exec(ctx, query, args...); err != nil {
		return err
	}

	return nil
}

func (r *Repo) getDevelopersIDs(ctx context.Context, conn pgxConn, developers []string) ([]int, error) {
	ids := make([]int, 0, len(developers))
	for _, developerName := range developers {
		id, cached := r.developersCache.get(developerName)
		if !cached {
			insertedID, err := r.insertAndCacheDeveloper(ctx, conn, developerName)
			if err != nil {
				return nil, err
			}
			id = insertedID
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *Repo) insertAndCacheDeveloper(ctx context.Context, conn pgxConn, developerName string) (int, error) {
	query, args, _ := sq.NewBuilder().
		Insert("developers").
		Columns("name").
		Values(developerName).
		Suffix("on conflict (name) do update set name = excluded.name returning id").
		ToSql()
	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	id, err := pgx.CollectOneRow(rows, pgx.RowTo[int])
	if err != nil {
		return 0, err
	}
	r.developersCache.set(developerName, id)
	return id, nil
}

func (r *Repo) getPublishersIDs(ctx context.Context, conn pgxConn, publishers []string) ([]int, error) {
	ids := make([]int, 0, len(publishers))
	for _, publisherName := range publishers {
		id, cached := r.publishersCache.get(publisherName)
		if !cached {
			insertedID, err := r.insertAndCachePublisher(ctx, conn, publisherName)
			if err != nil {
				return nil, err
			}
			id = insertedID
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *Repo) insertAndCachePublisher(ctx context.Context, conn pgxConn, publisherName string) (int, error) {
	query, args, _ := sq.NewBuilder().
		Insert("publishers").
		Columns("name").
		Values(publisherName).
		Suffix("on conflict (name) do update set name = excluded.name returning id").
		ToSql()
	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	id, err := pgx.CollectOneRow(rows, pgx.RowTo[int])
	if err != nil {
		return 0, err
	}
	r.publishersCache.set(publisherName, id)
	return id, nil
}

func (r *Repo) InsertApp(ctx context.Context, app *App) error {
	query, args, _ := sq.NewBuilder().
		Insert("apps_list").
		SetMap(map[string]any{
			"steam_id":      app.SteamID,
			"name":          app.Name,
			"last_modified": app.LastModified,
			"last_parsed":   app.LastParsed,
		}).
		ToSql()

	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repo) InsertApps(ctx context.Context, apps []App) error {
	builder := sq.NewBuilder().Insert("apps_list").Columns("steam_id", "name", "last_modified", "last_parsed")

	for _, app := range apps {
		builder = builder.Values(app.SteamID, app.Name, app.LastModified, app.LastParsed)
	}

	query, args, _ := builder.ToSql()

	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repo) UpdateTag(ctx context.Context, tag Tag) error {
	query, args, _ := sq.NewBuilder().
		Insert("tags").
		Columns("id", "name").
		Values(tag.ID, tag.Name).
		Suffix("on conflict (id) do update set name = excluded.name").
		ToSql()

	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	return nil
}
