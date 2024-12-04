load("github.com/SonarSource/cirrus-modules@v3", "load_features")
load("cirrus", "env", "fs", "yaml")

def main(ctx):
    return load_features(ctx)
