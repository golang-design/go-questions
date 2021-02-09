all:
	hugo
s:
	hugo server -D
clean:
	rm -rf public data