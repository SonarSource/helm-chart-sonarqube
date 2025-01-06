load("cirrus", "fs")


def build_tasks(ctx):
    tasks_env = fs.read(".cirrus/tasks_env.yml")
    tasks_templates = fs.read(".cirrus/tasks_templates.yml")
    tasks = fs.read(".cirrus/tasks.yml")
    tasks += fs.read(".cirrus/tasks_sonarqube.yml")
    tasks += fs.read(".cirrus/tasks_sonarqube_dce.yml")
    return tasks_env + tasks_templates + tasks
