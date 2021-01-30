#!/usr/bin/env python3

"""
Helmizer - Generates `kustomization.yaml` for your locally-rendered YAML manifests, such as from Helm templates
"""

import argparse
import logging
from os import path, walk
from sys import stdout

import yaml
from validators import url as validate_url


class Kustomization():
    def __init__(self, arguments):
        """Creates a kustomization.

        Args:
            arguments (dict): CLI arguments
        """

        # scaffolding
        self.arguments = arguments
        self.schema_defaults = {
            # 'apiVersion': 'kustomize.config.k8s.io/v1beta1',
            'kind': 'Kustomization',
        }
        self.yaml = dict(self.schema_defaults)

        # namespace
        self.api_version = self.get_api_version()

        # namespace
        self.namespace = self.get_namespace()

        # commonAnnotations
        self.common_annotations = self.get_common_annotations()

        # commonLabels
        self.common_labels = self.get_common_labels()

        # patchesStrategicMerge
        self.patches_strategic_merge = self.get_files(
            arguments, arguments.patches_strategic_merge, "patchesStrategicMerge")

        # resources
        self.resources = self.get_files(arguments, arguments.resources, "resources")

    def sort_keys(self):

        try:
            for array in "resources", "patchesStrategicMerge":
                self.yaml[array].sort()
        except:
            pass

    def print_kustomization(self):
        print(yaml.dump(self.yaml, sort_keys=self.arguments.sort_keys))

    def write_kustomization(self):
        if self.arguments.dry_run == False:
            self.arguments.kustomization_directory[0] = path.normpath(self.arguments.kustomization_directory[0])
            kustomization_file_path = str(f"{self.arguments.kustomization_directory[0]}/{self.arguments.kustomization_file_name}")
            with open(kustomization_file_path, 'w') as file:
                file.write(yaml.dump(self.yaml, sort_keys=self.arguments.sort_keys))
                logging.debug(f"Successfully wrote to file: {kustomization_file_path}")
        else:
            return

    def render_template(self):
        if self.arguments.sort_keys:
            self.sort_keys()
        self.print_kustomization()
        self.write_kustomization()

    def get_api_version(self):
        try:
            self.yaml["apiVersion"] = self.arguments.api_version
            logging.debug(f"apiVersion: {self.arguments.api_version}")
            return self.arguments.api_version
        except TypeError as e:
            logging.debug(e)
            pass

    def get_namespace(self):
        try:
            if len(self.arguments.namespace) > 0:
                self.yaml["namespace"] = self.arguments.namespace
                logging.debug(f"namespace: {self.arguments.namespace}")
                return self.arguments.namespace
        except TypeError as e:
            logging.debug(e)
            pass

    def get_common_annotations(self):
        try:
            if len(self.arguments.common_annotations) > 0:
                logging.debug(f"commonAnnotations: {self.arguments.common_annotations}")
                dict_common_annotations = dict()
                for annotation in self.arguments.common_annotations[0]:
                    list_annotation = annotation.split("=")
                    dict_common_annotations[list_annotation[0]] = list_annotation[1]
                self.yaml["commonAnnotations"] = dict_common_annotations
                return dict_common_annotations
        except TypeError as e:
            logging.debug(e)
            pass

    def get_common_labels(self):
        try:
            if len(self.arguments.common_labels) > 0:
                logging.debug(f"commonLabels: {self.arguments.common_labels}")
                dict_common_labels = dict()
                for label in self.arguments.common_labels[0]:
                    list_label = label.split("=")
                    dict_common_labels[list_label[0]] = list_label[1]
                self.yaml["commonLabels"] = dict_common_labels
                return dict_common_labels
        except TypeError as e:
            logging.debug(e)
            pass

    def get_files(self, arguments, paths, yaml_key):
        """Parses a path into a directory, file, or URL.

        Args:
            arguments ({dict}): CLI arguments
            paths ({list}): Paths to YAMLs
            yaml_key ({str}): YAML key such as 'resources', 'commonLabels', etc

        Returns:
            [list]: [description]
        """
        try:
            # normalize for OS
            for file_path in paths[0]:
                path.normpath(file_path)

            if len(paths[0]) > 0:
                list_final_target_paths = list()
                for target_path in paths[0]:

                    # walk directory
                    if path.isdir(target_path):
                        for (dirpath, _, filenames) in walk(target_path):
                            for file in filenames:
                                absolute_path = str(f"{dirpath}/{file}")
                                if arguments.resource_absolute_paths:
                                    list_final_target_paths.append(absolute_path)
                                else:
                                    str_relative_path = path.relpath(absolute_path, arguments.kustomization_directory[0])
                                    list_final_target_paths.append(str_relative_path)

                    # file
                    elif path.isfile(target_path):
                        absolute_path = path.abspath(target_path)
                        if arguments.resource_absolute_paths:
                            list_final_target_paths.append(
                                absolute_path)
                        else:
                            str_relative_path = path.relpath(
                                absolute_path, arguments.kustomization_directory[0])
                            list_final_target_paths.append(str_relative_path)

                    # url
                    elif validate_url(target_path):
                        list_final_target_paths.append(target_path)

                # wrap up
                self.yaml[yaml_key] = list()
                for target_path in list_final_target_paths:
                    self.yaml[yaml_key].append(target_path)

                return list_final_target_paths
        except TypeError as e:
            logging.debug(e)
            pass


def init_arg_parser():
    try:
        parser = argparse.ArgumentParser(prog='helmizer', description='Helmizer', formatter_class=argparse.ArgumentDefaultsHelpFormatter)

        optionals = parser.add_argument_group()
        optionals.add_argument("--apiVersion", dest='api_version', action='store', type=str, help='Specify the Kustomization \'apiVersion\'',
                               default="kustomize.config.k8s.io/v1beta1")
        optionals.add_argument("--commonAnnotations", dest='common_annotations', action='append', nargs='*',
                               help='Common Annotations where \'=\' is the assignment operator e.g linkerd.io/inject=enabled')
        optionals.add_argument("--commonLabels", dest='common_labels', action='append', nargs='*',
                               help='Common Labels where \'=\' is the assignment operator e.g labelname=labelvalue')
        optionals.add_argument("--debug", dest='debug', action='store_true', help='Enable debug logging', default=False)
        optionals.add_argument("--dry-run", dest='dry_run', action='store_true', help='Do not write to a file system.', default=False)
        optionals.add_argument("--kustomization-directory", dest='kustomization_directory', action='store', type=str,
                               nargs=1, help='Path to directory to contain the kustomization file',
                              required=True)
        optionals.add_argument("--kustomization-file-name", dest='kustomization_file_name', action='store',
                               type=str, help='options: \'kustomization.yaml\', \'kustomization.yml\', \'Kustomization\'',
                               default='kustomization.yaml')
        optionals.add_argument("--namespace", "-n", dest='namespace', action='store', type=str, help='Specify namespace in kustomization')
        optionals.add_argument("--patchesStrategicMerge", dest='patches_strategic_merge', action='append', nargs='*',
                               help='Path(s) to patch directories or files patchesStrategicMerge')
        optionals.add_argument("--resources", dest='resources', action='append', nargs='*',
                               help='Path(s) to resource directories or files')
        optionals.add_argument("--resource-absolute-paths", dest='resource_absolute_paths', action='append', nargs='*',
                               help='TODO')
        optionals.add_argument("--sort-keys", dest='sort_keys', action='store_true', help='Sort keys in arrays/lists',
                               default=False)
        optionals.add_argument('--version', action='version', version='v0.4.0')
        arguments = parser.parse_args()

        if arguments.debug:
            logging.basicConfig(level=logging.DEBUG, datefmt=None, stream=stdout,
                        format='[%(asctime)s %(levelname)s] %(message)s')
        else:
            logging.basicConfig(level=logging.INFO, datefmt=None, stream=stdout,
                        format='[%(asctime)s %(levelname)s] %(message)s')

        return arguments
    except argparse.ArgumentError as e:
        logging.error("Error parsing arguments")
        raise e


def main():
    arguments = init_arg_parser()
    kustomization = Kustomization(arguments)
    kustomization.render_template()


if __name__ == '__main__':
    try:
        main()
    except KeyboardInterrupt as e:
        raise e
    except SystemExit as e:
        raise e
