Final=/data/store/www/fosterphotos
Exec=photosite

all: dev

dev:
	go build

prod:
	go build -tags="PROD"

deploy: prod
	sudo $(Final)/$(Exec) stop
	rm -R $(Final)/lib
	rm -R $(Final)/template
	cp ./$(Exec) $(Final)/
	cp -R ./lib $(Final)/
	cp -R ./template $(Final)/
	sudo $(Final)/$(Exec) start

restart:
	sudo $(Final)/$(Exec) stop
	sudo $(Final)/$(Exec) start
