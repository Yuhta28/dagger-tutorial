package testing

bar: string

#dagger: {
	compute: [
		{
			do: "fetch-container"
			ref: "alpine"
		},
		{
			do: "exec"
			dir: "/"
			args: ["sh", "-c", "echo \(foo.bar)"]
		}
	]
	foo: bar: bar
}