load("github.com/SonarSource/cirrus-modules@v3", "load_features")
load("cirrus", "env", "fs", "yaml")
load(".cirrus/tasks.star", "build_tasks")


def main(ctx):
    tasks = build_tasks(ctx)
    return yaml.dumps(load_features(ctx)) + tasks
