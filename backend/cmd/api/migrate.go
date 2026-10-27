package main
import (
 "context"
 "embed"
 "fmt"
 "sort"
 "strings"
 "github.com/jackc/pgx/v5/pgxpool"
)
//go:embed migrations/*.sql
var migrationFiles embed.FS
func runMigrations(ctx context.Context,db *pgxpool.Pool)error{
 if _,err:=db.Exec(ctx,`CREATE TABLE IF NOT EXISTS schema_migrations(version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`);err!=nil{return fmt.Errorf("create migration table: %w",err)}
 entries,err:=migrationFiles.ReadDir("migrations");if err!=nil{return err}
 names:=make([]string,0,len(entries));for _,e:=range entries{if !e.IsDir()&&strings.HasSuffix(e.Name(),".sql"){names=append(names,e.Name())}};sort.Strings(names)
 for _,name:=range names{
  var applied bool;if err:=db.QueryRow(ctx,`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`,name).Scan(&applied);err!=nil{return err};if applied{continue}
  sqlBytes,err:=migrationFiles.ReadFile("migrations/"+name);if err!=nil{return err};tx,err:=db.Begin(ctx);if err!=nil{return err}
  if _,err=tx.Exec(ctx,string(sqlBytes));err==nil{_,err=tx.Exec(ctx,`INSERT INTO schema_migrations(version) VALUES($1)`,name)}
  if err!=nil{_=tx.Rollback(ctx);return fmt.Errorf("apply %s: %w",name,err)};if err=tx.Commit(ctx);err!=nil{return fmt.Errorf("commit %s: %w",name,err)}
 };return nil
}
