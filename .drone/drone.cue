kind: "pipeline"
name: "default"

#Step: {
  name: string,
  commands: [string],
  image: string | *"golang:1.19",
  volumes: [{name: "gopath", path: "/go"}],
  ...
}

steps: [...#Step]
steps: [{
  name: "download",
  commands: ["go mod download"],
}, {
  name: "lint",
  commands: ["make lint"],
  depends_on: ["download"],
}, {
  name: "test",
  commands: ["make test"],
  depends_on: ["download"],
},
]
