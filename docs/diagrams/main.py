from diagrams import Cluster
from diagrams import Diagram
from diagrams import Node
from diagrams.k8s.ecosystem import Helm
from diagrams.k8s.ecosystem import Kustomize
from diagrams.programming.flowchart import Action
from diagrams.programming.flowchart import InputOutput
from diagrams.programming.language import Bash
from diagrams.programming.language import Go


def main():
    with Diagram("./docs/diagrams/outputs/helmizer", show=False):
        with Cluster("Example Helmizer Flow"):
            config_file = InputOutput("helmizer.yaml")
            helmizer = Go("Helmizer")
            with Cluster("preCommands"):
                preCommands_1 = Helm("helm template")
                preCommands_2 = Bash("shell command")
            action = Action("Recursive file lookup")
            kustomization = Kustomize("Render\nkustomization.yaml")
            with Cluster("postCommands"):
                postCommands_1 = Bash("rm -rf \n./tests/go/away/file")
                postCommands_2 = Bash("pre-commit run -a")

            config_file >> helmizer >> [preCommands_1, preCommands_2] >> action >> kustomization >> [postCommands_1, postCommands_2]


if __name__ == "__main__":
    main()
