package main

func main() {
	a := App{}
	a.Initialize(
		"postgres",
		"snowplow",
		"snowplow")

	a.Run(":8080")
}