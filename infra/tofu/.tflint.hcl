# tflint for every unit (pre-commit runs it recursively).
plugin "terraform" {
  enabled = true
  preset  = "recommended"
}
