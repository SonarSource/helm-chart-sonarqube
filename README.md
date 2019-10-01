# Oteemo Charts Repository

This is the Oteemo helm charts repository.

### How It Works

GitHub Pages points to the `docs` folder so anything pushed to that directory will be publicly available.

### Process to add a chart to the repository

1. Create a branch for your new chart
1. Initialize new chart in the `charts` directory with `helm create mychart` or by copying in your work from outside
1. After chart development is done, run (at minimum) `helm lint mychart/` to validate yaml and templates
1. Don't forget to bump your chart version (if needed)
1. If your chart is ready then run the below commands to package, move and index your chart:

  ```console
  $ helm dependency update mychart (optional if you have requirements in requirements.yaml)
  $ helm package mychart
  $ mv mychart-0.1.0.tgz docs
  $ helm repo index docs --url https://oteemo.github.io/charts
  ```
1. Commit and push changes to branch
1. Open a pull request to merge your chart to the master branch.  Once it is merged it will be available publicly.

### Adding the chart Repository

`helm repo add oteemocharts https://oteemo.github.io/charts`
