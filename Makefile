run_local:
	go run main.go

run_migrate:
	go run ./db/migrate/migrate.go

run_db_migrate_down:
	go run db/migrate/migrate_down.go

run_seed:
	psql -U ahmadfarisi varion_development < db/seed.sql

run_docker:
	docker stop basecodeapiserver || true && docker rm basecodeapiserver || true
	docker build --tag basecode-api:dev .
	docker run --name basecodeapiserver -d -p 4000:4000 basecode-api:dev
