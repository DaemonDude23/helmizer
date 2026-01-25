from diagrams import Cluster
from diagrams import Diagram
from diagrams.k8s.ecosystem import Helm
from diagrams.k8s.ecosystem import Kustomize
from diagrams.programming.flowchart import Action
from diagrams.programming.flowchart import InputOutput
from diagrams.programming.language import Bash
from diagrams.programming.language import Go


def main():
    with Diagram("./docs/diagrams/outputs/helmizer", show=False):
        with Cluster("Inputs"):
            config_file = InputOutput("helmizer.yaml")
            cli_args = InputOutput("CLI args\n(overrides config)")
            ignore_file = InputOutput("helmizer.ignore")

        reconcile = Action("Reconcile config\n+ CLI overrides")
        helmizer = Go("Helmizer")
        with Cluster("preCommands (optional, ordered)"):
            pre_helm = Helm("helm template")
            pre_shell = Bash("any shell command")
        discover = Action("Walk resources/crds/\npatchesStrategicMerge\n(apply ignore list)")
        kustomization = Kustomize("Render\nkustomization.yaml")
        write_output = Action("Write output\n(or dry-run)")
        with Cluster("postCommands (optional, ordered)"):
            post_validate = Bash("validation/lint")
            post_shell = Bash("any shell command")
        output_file = InputOutput("kustomization.yaml\n(at kustomizationPath)")

        [config_file, cli_args] >> reconcile >> helmizer
        helmizer >> pre_helm >> pre_shell >> discover
        ignore_file >> discover
        discover >> kustomization >> write_output >> post_validate >> post_shell >> output_file


if __name__ == "__main__":
    main()
