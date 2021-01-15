#!/usr/bin/env python3
# -*- coding: utf8 -*-
"""
Helmizer - Generates `kustomization.yaml` for your locally-rendered YAML manifests, such as Helm templates
"""
import argparse
import logging
from os import pardir, path, walk
from sys import stdout

import yaml


class Kustomization():
    def __init__(self, arguments):

        # scaffolding
        self.arguments = arguments
        self.schema_defaults = {
            'apiVersion': 'kustomize.config.k8s.io/v1beta1',
            'kind': 'Kustomization',
        }
        self.yaml = dict(self.schema_defaults)

        # optional
        self.namespace = self.get_namespace()
        # patchesStrategicMerge
        self.patches_strategic_merge_paths = self.get_files(
            arguments, arguments.patches_strategic_merge_paths, "patchesStrategicMerge")
        # resources
        self.resources = self.get_files(
            arguments, arguments.resource_paths, "resources")
        # commonLabels
        self.common_labels = self.get_common_labels()

    def print_kustomization(self):
        print(f"{yaml.dump(self.yaml, sort_keys=True)}")

    def write_kustomization(self):
        if self.arguments.dry_run == False:
            self.arguments.kustomization_directory[0] = path.normpath(
                self.arguments.kustomization_directory[0])
            kustomization_directory = str(
                f"{self.arguments.kustomization_directory[0]}/{self.arguments.kustomization_file_name}")
            with open(kustomization_directory, 'w') as file:
                documents = yaml.dump(self.yaml, file, sort_keys=True)
                logging.info(
                    f"Successfully wrote to file: {kustomization_directory}")

    def render_template(self):
        self.print_kustomization()
        self.write_kustomization()

    def get_namespace(self):
        try:
            if len(self.arguments.namespace) > 0:
                self.yaml["namespace"] = self.arguments.namespace
                logging.debug(f"namespace: {self.arguments.namespace}")
                return self.arguments.namespace
        except TypeError as e:
            pass

    def get_common_labels(self):
        try:
            if len(self.arguments.common_labels) > 0:
                logging.debug(f"commonLabels: {self.arguments.common_labels}")
                dict_common_labels = dict()
                for count, label in enumerate(self.arguments.common_labels[0]):
                    list_label = label.split("=")
                    dict_common_labels[list_label[0]] = list_label[1]
                self.yaml["commonLabels"] = dict_common_labels
                return dict_common_labels
        except TypeError as e:
            pass

    def get_files(self, arguments, paths, yaml_key):
        try:
            for file_path in paths[0]:  # normalize for OS
                path.normpath(file_path)
            if len(paths[0]) > 0:
                list_final_target_paths = list()
                for target_path in paths[0]:
                    # path
                    if path.isdir(target_path):
                        for (dirpath, dirnames, filenames) in walk(target_path):
                            for file in filenames:
                                absolute_path = str(f"{dirpath}/{file}")
                                if arguments.resource_absolute_paths:
                                    list_final_target_paths.append(
                                        absolute_path)
                                else:
                                    str_relative_path = path.relpath(
                                        absolute_path, arguments.kustomization_directory[0])
                                    list_final_target_paths.append(
                                        str_relative_path)
                    # file
                    elif path.isfile(target_path):
                        absolute_path = path.abspath(target_path)
                        if arguments.resource_absolute_paths:
                            list_final_target_paths.append(
                                absolute_path)
                        else:
                            str_relative_path = path.relpath(
                                absolute_path, arguments.kustomization_directory[0])
                            list_final_target_paths.append(
                                str_relative_path)
                    # TODO url

                # wrap up
                self.yaml[yaml_key] = list()
                for target_path in list_final_target_paths:
                    self.yaml[yaml_key].append(target_path)
                return list_final_target_paths
        except TypeError as e:
            pass


def init_arg_parser():
    try:
        parser = argparse.ArgumentParser(prog='helmizer', description='Helmizer',
                                         formatter_class=argparse.ArgumentDefaultsHelpFormatter)

        optionals = parser.add_argument_group()
        optionals.add_argument("--namespace", "-n", dest='namespace', action='store', type=str, help='')
        optionals.add_argument("--kustomization-file-name", dest='kustomization_file_name', action='store',
                               type=str, help='options: `kustomization.yaml`, kustomization.yml, `Kustomization`',
                               default='kustomization.yaml')
        optionals.add_argument("--debug", dest='debug', action='store_true', help='', default=False)
        optionals.add_argument("--dry-run", dest='dry_run', action='store_true', help='', default=False)
        optionals.add_argument("--patches-strategic-merge-paths", dest='patches_strategic_merge_paths', action='append', nargs='*', help='')
        optionals.add_argument("--resource-paths", dest='resource_paths', action='append', nargs='*', help='')
        optionals.add_argument("--resource-absolute-paths", dest='resource_absolute_paths', action='append', nargs='*', help='')
        optionals.add_argument("--common-labels", dest='common_labels', action='append', nargs='*', help='')

        required = parser.add_argument_group()
        required.add_argument("--kustomization-directory", dest='kustomization_directory',
                              action='store', type=str, nargs=1, help='', required=True)

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
    # kustomize
    kustomization = Kustomization(arguments)
    kustomization.render_template()


if __name__ == '__main__':
    try:
        main()
    except KeyboardInterrupt as e:
        raise e
    except SystemExit as e:
        raise e
