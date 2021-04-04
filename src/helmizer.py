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
        list_patches_strategic_merge = self.get_files(arguments, 'patchesStrategicMerge')
        if list_patches_strategic_merge:
            self.yaml['patchesStrategicMerge'] = list_patches_strategic_merge

        # resources
        list_resources = self.get_files(arguments, 'resources')
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
        str_kustomization_dir = str()

        if self.arguments.kustomization_dir:
            str_kustomization_dir = self.arguments.kustomization_dir
        else:
            try:
                str_kustomization_dir = self.helmizer_config['helmizer']['kustomization-directory'].get(str)
            except KeyError:
                str_kustomization_dir = getcwd()

        # identify kustomization file name
        # TODO allow kustomization of file name
        str_kustomization_file_name = str()
        try:
            str_kustomization_file_name = self.helmizer_config['helmizer']['kustomization-file-name'].get(str)
        except KeyError:
            str_kustomization_file_name = 'kustomization.yaml'

        # write to file
        try:
            kustomization_file_path = path.normpath(f'{str_kustomization_dir}/{str_kustomization_file_name}')
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


    def get_files(self, arguments, key):
        list_target_paths = list()
        list_final_target_paths = list()
        str_kustomization_path = str()

        try:
            # test if the key to configure is even defined in input helmizer config
            list_kustomization_children = self.helmizer_config['kustomize'][key].get(list)

            if arguments.kustomization_dir:
                str_kustomization_path = path.abspath(arguments.kustomization_dir)
            else:
                str_kustomization_path = path.abspath(self.helmizer_config['helmizer']['kustomization-directory'].get(str))

            if len(list_kustomization_children) > 0:
                for target_path in list_kustomization_children:
                    str_child_path = path.abspath(path.join(str_kustomization_path, target_path))

                    # walk directory
                    if path.isdir(str_child_path):
                        for (dirpath, _, filenames) in walk(str_child_path):
                            for filename in filenames:
                                list_target_paths.append(path.join(dirpath, filename))

                    # file
                    elif path.isfile(str_child_path):
                        list_target_paths.append(str_child_path)

                    # url
                    elif validate_url(str_child_path):
                        list_target_paths.append(str_child_path)

                # convert absolute paths into paths relative to the kustomization directory
                for final_target_path in list_target_paths:
                    list_final_target_paths.append(path.relpath(final_target_path, str_kustomization_path))

                return list_final_target_paths

        except NotFoundError:
            logging.debug(f'key not found: {key}')
            pass
        except TypeError:
            pass


    def run_subprocess(self, arguments):
        subprocess_working_directory = str()
        if arguments.kustomization_dir:
            subprocess_working_directory = path.abspath(path.normpath(arguments.kustomization_dir))
        else:
            subprocess_working_directory = path.abspath(path.normpath(self.helmizer_config['helmizer']['kustomization-directory'].get(str)))
        logging.debug(f'Subprocess working directory: {subprocess_working_directory}')


        list_command_string = list()
        for config_command in self.helmizer_config['helmizer']['commandSequence']:
            try:

                # construct command(s)
                if config_command['command']:
                    command = config_command['command'].get(str)
                    if config_command['args']:
                        args = ' '.join(config_command['args'].get(list))  # combine list elements into space-delimited
                        list_command_string.append(f'{command} {args}')
                    else:
                        list_command_string.append(f'{command}')

                # execute
                for command in list_command_string:
                    logging.debug(f"creating subprocess: \'{command}\'")
                    if arguments.quiet:
                        subprocess.run(f'{command}', shell=True, check=True, stdout=subprocess.DEVNULL,
                                    stderr=subprocess.DEVNULL, text=True, cwd=subprocess_working_directory)
                    else:
                        subprocess.run(f'{command}', shell=True, check=True, text=True, cwd=subprocess_working_directory)

            except NotFoundError as e:
                pass


def init_arg_parser():
    try:
        parser = argparse.ArgumentParser(prog='helmizer', description='Helmizer', formatter_class=argparse.ArgumentDefaultsHelpFormatter)

        optionals = parser.add_argument_group()
        optionals.add_argument('--debug', dest='debug', action='store_true', help='Enable debug logging', default=False)
        optionals.add_argument('--dry-run', dest='dry_run', action='store_true', help='Do not write to a file system.', default=False)
        optionals.add_argument('--helmizer-directory', dest='helmizer_directory', action='store', type=str,
                               help='Override helmizer file path')
        optionals.add_argument('--kustomization-directory', dest='kustomization_dir', action='store', type=str,
                               help='Set path containing kustomization')
        optionals.add_argument('--quiet', '-q', dest='quiet', action='store_true', help='Quiet output from subprocesses',
                               default=False)
        optionals.add_argument('--version', action='version', version='v0.5.2')
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
    str_helmizer_config_path = str()
    if arguments.helmizer_directory:
        str_helmizer_config_path = arguments.helmizer_directory
    else:
        str_helmizer_config_path = getcwd()

    config = confuse.Configuration('helmizer', __name__)
    try:
        try:
            # assume file name is helmizer.yaml
            logging.debug(f'Trying helmizer config path from config: {str_helmizer_config_path}/helmizer.yaml')
            config.set_file(f'{str_helmizer_config_path}/helmizer.yaml')
        except KeyError:
            if str_helmizer_config_path:
                logging.debug(f'Trying helmizer config path from argument: {str_helmizer_config_path}')
                config.set_file(path.normpath(str_helmizer_config_path))
        finally:
            logging.debug(f'parsed config: {config}')
    except confuse.exceptions.ConfigReadError:
        # no config file found. Give up
        return dict()

    try:
        validate_helmizer_config_version(config['helmizer']['version'].get(str))
    except KeyError:
        logging.debug('Unable to validate version')

    return config


def main():
    arguments = init_arg_parser()
    helmizer_config = init_helmizer_config(arguments)
    kustomization = Kustomization(helmizer_config, arguments)
    kustomization.run_subprocess(arguments)
    kustomization.render_template()


if __name__ == '__main__':
    try:
        main()
    except KeyboardInterrupt or SystemExit:
        exit(1)
