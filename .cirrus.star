load("github.com/SonarSource/cirrus-modules@v2", "load_features")

def main(ctx):
    return load_features(ctx, aws=dict(env_type="dev", cluster_name="CirrusCI-4-pr-83", subnet_id="subnet-02074adb01e2afc6d"))
