load("cirrus", "fs")


def build_tasks(ctx):
    tasks_env = fs.read(".cirrus/tasks_env.yml")
    tasks_templates = fs.read(".cirrus/tasks_templates.yml")
    tasks = fs.read(".cirrus/tasks.yml")
    tasks += fs.read(".cirrus/tasks_sonarqube.yml")
    tasks += fs.read(".cirrus/tasks_sonarqube_dce.yml")
    tasks += fs.read(".cirrus/tasks_gcp_marketplace.yml")

    # The release task depends on some sonarqube and sonarqube_dce tasks,
    # therefore it MUST be loaded AFTER tasks_sonarqube.yml and tasks_sonarqube_dce.yml
    tasks += fs.read(".cirrus/tasks_release.yml")

    return tasks_env + tasks_templates + tasks
