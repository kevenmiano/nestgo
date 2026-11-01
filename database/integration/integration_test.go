package integrationtests

import (
 "context"
 "database/sql"
 "errors"
 "os"
 "testing"
 "time"

 "github.com/go-sql-driver/mysql"
 nestdb "github.com/kevenmiano/nestgo/database"
 "github.com/kevenmiano/nestgo/database/sqlgen"
 nestsqlx "github.com/kevenmiano/nestgo/database/sqlx"
 "github.com/jmoiron/sqlx"
 _ "github.com/lib/pq"
)

func TestRealDatabases(t *testing.T) {
 for _, tc := range []struct {
  name, driver, env string
  dialect sqlgen.Dialect
 }{
  {"postgres", "postgres", "NESTGO_TEST_POSTGRES_DSN", sqlgen.PostgreSQL},
  {"mysql", "mysql", "NESTGO_TEST_MYSQL_DSN", sqlgen.Question},
 } {
  t.Run(tc.name,func(t *testing.T) {
   dsn:=os.Getenv(tc.env)
   if dsn=="" {t.Skipf("%s not configured; mandatory in CI",tc.env)}
   if tc.driver=="mysql" {
    config,err:=mysql.ParseDSN(dsn)
    if err!=nil {t.Fatal(err)}
    config.MultiStatements=false
    dsn=config.FormatDSN()
   }
   db,err:=sql.Open(tc.driver,dsn)
   if err!=nil {t.Fatal(err)}
   defer db.Close()
   ctx,cancel:=context.WithTimeout(context.Background(),20*time.Second)
   defer cancel()
   if err:=db.PingContext(ctx);err!=nil {t.Fatalf("real database unavailable: %v",err)}
   store,err:=nestdb.Native(db)
   if err!=nil {t.Fatal(err)}
   // CREATE and DELETE statements are static; user input only goes through placeholders.
   if _,err:=store.Exec(ctx,"CREATE TABLE IF NOT EXISTS nestgo_security_users (id INTEGER PRIMARY KEY, name VARCHAR(255) NOT NULL)");err!=nil {t.Fatal(err)}
   if _,err:=store.Exec(ctx,"DELETE FROM nestgo_security_users");err!=nil {t.Fatal(err)}
   defer db.ExecContext(context.Background(),"DELETE FROM nestgo_security_users")

   payload:="' OR 1=1; DROP TABLE nestgo_security_users; --"
   insert,args,err:=sqlgen.Bind("INSERT INTO nestgo_security_users (id,name) VALUES (:id,:name)",map[string]any{"id":1,"name":payload},tc.dialect)
   if err!=nil {t.Fatal(err)}
   if _,err:=store.Exec(ctx,insert,args...);err!=nil {t.Fatal(err)}
   query,args,err:=sqlgen.Bind("SELECT name FROM nestgo_security_users WHERE id=:id",map[string]any{"id":1},tc.dialect)
   if err!=nil {t.Fatal(err)}
   var got string
   if err:=store.QueryRow(ctx,query,args...).Scan(&got);err!=nil {t.Fatal(err)}
   if got!=payload {t.Fatalf("injected value was changed: %q",got)}
   attack,args,err:=sqlgen.Bind("SELECT COUNT(*) FROM nestgo_security_users WHERE name=:name",map[string]any{"name":payload},tc.dialect)
   if err!=nil {t.Fatal(err)}
   var count int
   if err:=store.QueryRow(ctx,attack,args...).Scan(&count);err!=nil || count!=1 {t.Fatalf("value should be matched literally: count=%d err=%v",count,err)}
   // A control query must still work, proving that the attack did not drop the table.
   if err:=store.QueryRow(ctx,"SELECT COUNT(*) FROM nestgo_security_users").Scan(&count);err!=nil || count!=1 {t.Fatalf("table damaged: count=%d err=%v",count,err)}

   sentinel:=errors.New("force rollback")
   err=nestdb.WithinTransaction(ctx,db,nil,func(tx *nestdb.Store)error {
    ins,values,e:=sqlgen.Bind("INSERT INTO nestgo_security_users (id,name) VALUES (:id,:name)",map[string]any{"id":2,"name":"temporary"},tc.dialect)
    if e!=nil {return e}
    _,e=tx.Exec(ctx,ins,values...)
    if e!=nil {return e}
    return sentinel
   })
   if !errors.Is(err,sentinel) {t.Fatalf("rollback returned %v",err)}
   if err:=store.QueryRow(ctx,"SELECT COUNT(*) FROM nestgo_security_users").Scan(&count);err!=nil || count!=1 {t.Fatalf("rollback failed: %d %v",count,err)}

   err=nestdb.WithinTransaction(ctx,db,nil,func(tx *nestdb.Store)error {
    ins,values,e:=sqlgen.Bind("INSERT INTO nestgo_security_users (id,name) VALUES (:id,:name)",map[string]any{"id":3,"name":"committed"},tc.dialect)
    if e!=nil {return e}
    _,e=tx.Exec(ctx,ins,values...)
    return e
   })
   if err!=nil {t.Fatal(err)}
   if err:=store.QueryRow(ctx,"SELECT COUNT(*) FROM nestgo_security_users").Scan(&count);err!=nil || count!=2 {t.Fatalf("commit failed: %d %v",count,err)}

   xdb:=sqlx.NewDb(db,tc.driver)
   adapter,err:=nestsqlx.New(xdb)
   if err!=nil {t.Fatal(err)}
   var user struct {ID int `db:"id"`; Name string `db:"name"`}
   // sqlx is tested against the real db, not a mock or method stub.
   sq:=xdb.Rebind("SELECT id,name FROM nestgo_security_users WHERE id=?")
   if err:=adapter.Get(ctx,&user,sq,1);err!=nil {t.Fatal(err)}
   if user.Name!=payload {t.Fatalf("sqlx mapping returned %q",user.Name)}
   var all []struct {ID int `db:"id"`; Name string `db:"name"`}
   if err:=adapter.Select(ctx,&all,"SELECT id,name FROM nestgo_security_users ORDER BY id");err!=nil || len(all)!=2 {t.Fatalf("sqlx select %d %v",len(all),err)}
  })
 }
}
