#!/usr/bin/env python3

"""
Helmizer - Generates `kustomization.yaml` for your locally-rendered YAML manifests,
such as from Helm templates or plain YAML manifests
"""

import argparse
import logging
from os import path, walk, getcwd
from sys import stdout
from typing import Type
import confuse
from confuse.exceptions import NotFoundError
import subprocess

import yaml
from validators import url as validate_url


class Kustomization():
    def __init__(self, helmizer_config, arguments):
        self.helmizer_config = helmizer_config
        self.arguments = arguments
        self.yaml = dict()

        # apiVersion
        self.yaml['apiVersion'] = self.get_api_version()

        # kind
        self.yaml['kind'] = 'Kustomization'

        # namespace
        str_namespace = self.get_namespace()
        if str_namespace:
            self.yaml['namespace'] = str_namespace

        # commonAnnotations
        dict_common_annotations = self.get_common_annotations()
        if dict_common_annotations:
            self.yaml['commonAnnotations'] = dict_common_annotations

        # commonLabels
        dict_get_common_labels = self.get_common_labels()
        if dict_get_common_labels:
            self.yaml['commonLabels'] = dict_get_common_labels

        # patchesStrategicMerge
        list_patches_strategic_merge = self.get_files('patchesStrategicMerge')
        if list_patches_strategic_merge:
            self.yaml['patchesStrategicMerge'] = list_patches_strategic_merge

        # resources
        list_resources = self.get_files('resources')
        if list_resources:
            self.yaml['resources'] = list_resources


    def sort_keys(self):
        try:
            self.helmizer_config['helmizer']['sort-keys'].get(bool)
            for array in 'resources', 'patchesStrategicMerge':
                self.yaml[array].sort()
        except KeyError:
            pass

    def print_kustomization(self):
        try:
            print(yaml.dump(self.yaml, sort_keys=False))
        except KeyError:
            print(yaml.dump(self.yaml, sort_keys=False))


    def write_kustomization(self):
        # identify kustomization file's parent directory
        str_kustomization_directory = str()
        try:
            str_kustomization_directory = self.helmizer_config['helmizer']['kustomization-directory'].get(str)
        except KeyError:
            str_kustomization_directory = '.'

        # identify kustomization file name
        str_kustomization_file_name = str()
        try:
            str_kustomization_file_name = self.helmizer_config['helmizer']['kustomization-file-name'].get(str)
        except KeyError:
            str_kustomization_file_name = 'kustomization.yaml'

        # write to file
        try:
            kustomization_file_path = path.normpath(f'{str_kustomization_directory}/{str_kustomization_file_name}')
            with open(kustomization_file_path, 'w') as file:
                file.write(yaml.dump(self.yaml))
                logging.debug(f'Successfully wrote to file: {kustomization_file_path}')
        except IsADirectoryError as e:
            raise e
        except TypeError:
            pass


    def render_template(self):
        self.sort_keys()
        self.print_kustomization()
        self.write_kustomization()

    def get_api_version(self):
        str_api_version = str()
        try:
            str_api_version = self.helmizer_config['kustomize']['apiVersion'].get(str)
        except NotFoundError:
            str_api_version = 'kustomize.config.k8s.io/v1beta1'
        finally:
            logging.debug(f'apiVersion: {str_api_version}')
            return str_api_version


    def get_namespace(self):
        str_namespace = str()
        try:
            if len(self.helmizer_config['kustomize']['namespace'].get(str)) > 0:
                str_namespace = self.helmizer_config['kustomize']['namespace'].get(str)
                logging.debug(f'namespace: {str_namespace}')
        except TypeError:
            pass
        finally:
            return str_namespace


    def get_common_annotations(self):
        dict_common_annotations = dict()
        try:
            if len(self.helmizer_config['kustomize']['commonAnnotations'].get(dict)) > 0:
                dict_common_annotations = dict(self.helmizer_config['kustomize']['commonAnnotations'].get(dict))
                logging.debug(f'commonAnnotations: {dict_common_annotations}')
        except TypeError:
            pass
        finally:
            return dict_common_annotations

    def get_common_labels(self):
        dict_common_labels = dict()
        try:
            if len(self.helmizer_config['kustomize']['commonLabels'].get(dict)) > 0:
                dict_common_labels = dict(self.helmizer_config['kustomize']['commonLabels'].get(dict))
                logging.debug(f'commonLabels: {dict_common_labels}')
        except TypeError:
            pass
        finally:
            return dict_common_labels

    def get_files(self, key):
        try:
            paths = self.helmizer_config['kustomize'][key].get(list)

            if len(paths) > 0:
                list_final_target_paths = list()
                for target_path in paths:

                    # walk directory
                    if path.isdir(target_path):
                        for (dirpath, _, filenames) in walk(target_path):
                            for file in filenames:
                                absolute_path = path.normpath(f'{dirpath}/{file}')
                                # TODO fix this
                                if len(self.helmizer_config['helmizer']['resource-absolute-paths'].get(list)) > 0:
                                    if len(self.helmizer_config['helmizer']['resource-absolute-paths'].get(list)) > 0:
                                        list_final_target_paths.append(absolute_path)
                                else:
                                    if self.helmizer_config['helmizer']['kustomization-directory']:
                                        str_relative_path = path.relpath(absolute_path, self.helmizer_config['helmizer']['kustomization-directory'].get(str))
                                    list_final_target_paths.append(str_relative_path)

                    # file
                    elif path.isfile(target_path):
                        absolute_path = path.abspath(target_path)
                        if self.helmizer_config['helmizer']['resource-absolute-paths'].get(bool):
                            list_final_target_paths.append(absolute_path)
                        else:
                            str_relative_path = path.relpath(absolute_path, getcwd())
                            list_final_target_paths.append(str_relative_path)

                    # url
                    elif validate_url(target_path):
                        list_final_target_paths.append(target_path)

                return list_final_target_paths
        except NotFoundError:
            pass
        except TypeError:
            pass


def run_subprocess(arguments, command_string):
    logging.debug(f"creating subprocess: \'{command_string}\'")
    # TODO make it respect this arg
    if arguments.quiet:
        subprocess.run(f'{command_string}', shell=True, check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, text=False)
    else:
        subprocess.run(f'{command_string}', shell=True, check=True, text=True)


def init_arg_parser():
    try:
        parser = argparse.ArgumentParser(prog='helmizer', description='Helmizer', formatter_class=argparse.ArgumentDefaultsHelpFormatter)

        optionals = parser.add_argument_group()
        optionals.add_argument('--debug', dest='debug', action='store_true', help='Enable debug logging', default=False)
        optionals.add_argument('--dry-run', dest='dry_run', action='store_true', help='Do not write to a file system.', default=False)
        optionals.add_argument('--helmizer-config-path', dest='helmizer_config_path', action='store', type=str,
                               help='Override helmizer file path. Default = \'$KUSTOMIZATION_PATH/helmizer.yaml\'',
                               default=getcwd())
        optionals.add_argument('--quiet', '-q', dest='quiet', action='store_true', help='Quiet output from subcommands',
                               default=False)
        optionals.add_argument('--version', action='version', version='v0.5.0')
        arguments = parser.parse_args()

        if arguments.quiet:
            logging.basicConfig(level=logging.INFO, datefmt=None, stream=None,
                        format='[%(asctime)s %(levelname)s] %(message)s')
        if arguments.debug:
            logging.basicConfig(level=logging.DEBUG, datefmt=None, stream=stdout,
                        format='[%(asctime)s %(levelname)s] %(message)s')
        else:
            logging.basicConfig(level=logging.INFO, datefmt=None, stream=stdout,
                        format='[%(asctime)s %(levelname)s] %(message)s')

        return arguments
    except argparse.ArgumentError as e:
        logging.error('Error parsing arguments')
        raise e


def validate_helmizer_config_version(helmizer_config_version):
    # TODO actually validate it
    logging.debug(f'validating helmizer config version: {helmizer_config_version}')


def init_helmizer_config(arguments):
    config = confuse.Configuration('helmizer', __name__)
    try:
        try:
            logging.debug(f'Trying helmizer config path from config: {arguments.helmizer_config_path}/helmizer.yaml')
            config.set_file(f'{arguments.helmizer_config_path}/helmizer.yaml')
            logging.debug(f'parsed config: {config}')
        except KeyError:
            if arguments.helmizer_config_path:
                logging.debug(f'Trying helmizer config path from argument: {arguments.helmizer_config_path}')
                config.set_file(path.normpath(arguments.helmizer_config_path))
                logging.debug(f'parsed config: {config}')
    except confuse.exceptions.ConfigReadError:
        # no config file found. Give up
        return dict()

    try:
        validate_helmizer_config_version(config['helmizer']['version'].get(str))
    except KeyError:
        logging.debug('Unable to validate version')

    # TODO give it its own function
    for config_command in config['helmizer']['commandSequence']:
        try:
            if config_command['command']:
                command = config_command['command'].get(str)
                if config_command['args']:
                    args = ' '.join(config_command['args'].get(list))  # combine list elements into space-delimited
                    run_subprocess(arguments, f'{command} {args}')
                else:
                    run_subprocess(command)
        except NotFoundError as e:
            pass

    return config


def main():
    arguments = init_arg_parser()
    helmizer_config = init_helmizer_config(arguments)
    kustomization = Kustomization(helmizer_config, arguments)
    kustomization.render_template()


if __name__ == '__main__':
    try:
        main()
    except KeyboardInterrupt or SystemExit:
        exit(1)
