#!/usr/bin/env python3

"""
Helmizer - Generates `kustomization.yaml` for your locally-rendered YAML manifests,
such as from Helm templates or plain YAML manifests
"""

import argparse
import logging
from os import path, walk
from sys import stdout
import confuse
from confuse.exceptions import NotFoundError
import subprocess
import json

import yaml
from validators import url as validate_url


class Kustomization:
    def __init__(self, helmizer_config, arguments):
        self.helmizer_config = helmizer_config
        self.arguments = arguments
        self.yaml = dict()

        # apiVersion
        # TODO put this into get_str() ?
        self.yaml["apiVersion"] = self.get_api_version()

        # kind
        self.yaml["kind"] = "Kustomization"

        # get strings
        for key in ["namePrefix", "nameSuffix", "namespace"]:
            try:
                str_key = self.get_str(key)
                if str_key:
                    self.yaml[key] = str_key
            except NotFoundError:
                pass

        # get lists
        for key in ["configMapGenerator", "images", "patchesJson6902", "replicas", "replacements", "vars"]:
            try:
                list_key = self.get_list(key)
                if list_key:
                    self.yaml[key] = list_key
            except NotFoundError:
                pass

        # get dicts
        for key in ["commonAnnotations", "commonLabels", "generatorOptions", "openapi"]:
            try:
                dict_key = self.get_dict(key)
                if dict_key:
                    self.yaml[key] = dict_key
            except NotFoundError:
                pass

        # get files
        for key in ["crds", "components", "patchesStrategicMerge", "resources"]:
            try:
                list_key = self.get_files(arguments, key)
                if list_key:
                    self.yaml[key] = list_key
            except NotFoundError:
                pass

    def sort_keys(self):
        try:
            self.helmizer_config["helmizer"]["sort-keys"].get(bool)
            for array in ["crds", "components", "patchesStrategicMerge", "resources"]:
                self.yaml[array].sort()
        except KeyError:
            pass

    def print_kustomization(self):
        try:
            print(yaml.dump(self.yaml, sort_keys=False))
        except KeyError:
            print(yaml.dump(self.yaml, sort_keys=False))

    def write_kustomization(self, arguments):
        try:
            if self.helmizer_config["helmizer"]["dry-run"].get(bool) or arguments.dry_run:
                logging.debug("Performing dry-run, not writing to a file system")
        except NotFoundError:
            # identify kustomization file's parent directory
            str_kustomization_directory = path.dirname(path.abspath(path.normpath(arguments.helmizer_config)))

            # identify kustomization file name
            str_kustomization_file_name = str()
            try:
                str_kustomization_file_name = self.helmizer_config["helmizer"]["kustomization-file-name"].get(str)
            except NotFoundError:
                str_kustomization_file_name = "kustomization.yaml"

            # write to file
            try:
                kustomization_file_path = path.normpath(f"{str_kustomization_directory}/{str_kustomization_file_name}")
                with open(kustomization_file_path, "w") as file:
                    file.write(yaml.dump(self.yaml))
                    logging.debug(f"Successfully wrote to file: {path.abspath(kustomization_file_path)}")
            except IsADirectoryError as e:
                raise e
            except TypeError:
                pass

    def render_template(self, arguments):
        logging.debug("Rendering template")
        self.sort_keys()
        self.print_kustomization()
        self.write_kustomization(arguments)

    def get_api_version(self):
        str_api_version = str()
        try:
            str_api_version = self.helmizer_config["kustomize"]["apiVersion"].get(str)
        except NotFoundError:
            str_api_version = "kustomize.config.k8s.io/v1beta1"
        finally:
            logging.debug(f"apiVersion: {str_api_version}")
            return str_api_version

    def get_files(self, arguments, key):
        list_target_paths = list()
        list_final_target_paths = list()
        str_kustomization_path = str()

        try:
            # test if the key to configure is even defined in input helmizer config
            list_kustomization_children = self.helmizer_config["kustomize"][key].get(list)
            str_kustomization_directory = str()
            try:
                str_kustomization_directory = self.helmizer_config["helmizer"]["kustomization-directory"].get(str)
            except NotFoundError:
                str_kustomization_directory = "."

            str_kustomization_path = path.dirname(
                path.abspath(
                    path.normpath(
                        path.join(
                            arguments.helmizer_config,
                            str_kustomization_directory,
                        )
                    )
                )
            )

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

                # remove any ignored files
                try:
                    for ignore in self.helmizer_config["helmizer"]["ignore"].get(list):
                        logging.debug(f"Removing ignored file from final list: {ignore}")
                        list_final_target_paths.remove(ignore)
                except ValueError:
                    pass
                except NotFoundError:
                    pass

                return list_final_target_paths

        except NotFoundError:
            logging.debug(f"key not found: {key}")
            pass
        except KeyError:
            logging.debug(f"key not found: {key}")
            pass
        except TypeError:
            pass

    def get_dict(self, key):
        dict_raw_yaml = dict()
        try:
            if self.helmizer_config["kustomize"][key].get(dict):
                dict_raw_yaml = json.loads(json.dumps(self.helmizer_config["kustomize"][key].get(dict)))
                logging.debug(f"{key}: {dict_raw_yaml}")
        except NotFoundError:
            logging.debug(f"key not found: {key}")
            pass
        except KeyError:
            logging.debug(f"key not found: {key}")
            pass
        except TypeError:
            pass
        return dict_raw_yaml

    def get_str(self, key):
        str_raw_yaml = str()
        try:
            if self.helmizer_config["kustomize"][key].get(str):
                str_raw_yaml = json.loads(json.dumps(self.helmizer_config["kustomize"][key].get(str)))
                logging.debug(f"{key}: {str_raw_yaml}")
        except NotFoundError:
            logging.debug(f"key not found: {key}")
            pass
        except KeyError:
            logging.debug(f"key not found: {key}")
            pass
        except TypeError:
            pass
        return str_raw_yaml

    def get_list(self, key):
        list_raw_yaml = list()
        try:
            if len(self.helmizer_config["kustomize"][key].get(list)) > 0:
                list_raw_yaml = json.loads(json.dumps(self.helmizer_config["kustomize"][key].get(list)))
                logging.debug(f"{key}: {list_raw_yaml}")
        except NotFoundError:
            logging.debug(f"key not found: {key}")
            pass
        except KeyError:
            logging.debug(f"key not found: {key}")
            pass
        except TypeError:
            pass
        return list_raw_yaml


def run_subprocess(helmizer_config, arguments):
    subprocess_working_directory = path.dirname(path.abspath(path.normpath(arguments.helmizer_config)))
    logging.debug(f"Subprocess working directory: {subprocess_working_directory}")
    list_command_string = list()
    try:
        for config_command in helmizer_config["helmizer"]["commandSequence"]:
            # construct command(s)
            if config_command["command"]:
                command = config_command["command"].get(str)
                if config_command["args"]:
                    args = " ".join(config_command["args"].get(list))  # combine list elements into space-delimited
                    list_command_string.append(f"{command} {args}")
                else:
                    list_command_string.append(f"{command}")

        # execute
        for command in list_command_string:
            if arguments.quiet:
                logging.debug(f"creating subprocess: '{command}'")
                subprocess.run(
                    f"{command}",
                    shell=True,
                    check=True,
                    stdout=subprocess.DEVNULL,
                    stderr=subprocess.DEVNULL,
                    text=True,
                    cwd=subprocess_working_directory,
                )
            else:
                logging.info(f"creating subprocess: '{command}'")
                subprocess.run(
                    f"{command}",
                    shell=True,
                    check=True,
                    text=True,
                    cwd=subprocess_working_directory,
                )
    except NotFoundError:
        pass


def init_arg_parser():
    try:
        parser = argparse.ArgumentParser(
            prog="helmizer",
            description="Helmizer",
            formatter_class=argparse.ArgumentDefaultsHelpFormatter,
        )
        args = parser.add_argument_group()
        args.add_argument(
            "--debug",
            dest="debug",
            action="store_true",
            help="enable debug logging",
            default=False,
        )
        args.add_argument(
            "--dry-run",
            dest="dry_run",
            action="store_true",
            help="do not write to a file system",
            default=False,
        )
        args.add_argument(
            "--skip-commands",
            dest="skip_commands",
            action="store_true",
            help="skip executing commandSequence, just generate kustomization file",
            default=False,
        )
        args.add_argument(
            "--quiet",
            "-q",
            dest="quiet",
            action="store_true",
            help="quiet output from subprocesses",
            default=False,
        )
        args.add_argument("--version", "-v", action="version", version="v0.9.0")
        args.add_argument(
            "helmizer_config",
            action="store",
            type=str,
            help="path to helmizer config file",
        )
        arguments = parser.parse_args()

        if arguments.quiet:
            logging.basicConfig(
                level=logging.INFO,
                datefmt=None,
                stream=None,
                format="[%(asctime)s %(levelname)s] %(message)s",
            )
        if arguments.debug:
            logging.basicConfig(
                level=logging.DEBUG,
                datefmt=None,
                stream=stdout,
                format="[%(asctime)s %(levelname)s] %(message)s",
            )
        else:
            logging.basicConfig(
                level=logging.INFO,
                datefmt=None,
                stream=stdout,
                format="[%(asctime)s %(levelname)s] %(message)s",
            )

        return arguments
    except argparse.ArgumentError as e:
        logging.error("Error parsing arguments")
        raise e


def validate_helmizer_config_version(helmizer_config_version):
    # TODO actually validate it
    logging.debug(f"validating helmizer config version: {helmizer_config_version}")


def init_helmizer_config(arguments):
    str_helmizer_config_path = arguments.helmizer_config

    config = confuse.Configuration("helmizer", __name__)
    try:
        if str_helmizer_config_path:
            logging.debug(f"Trying helmizer config path from argument: {str_helmizer_config_path}")
            config.set_file(path.normpath(str_helmizer_config_path))
            logging.debug(f"parsed config: {config}")

    # no config file found. Give up
    except confuse.exceptions.ConfigReadError:
        logging.error(f"Unable to locate helmizer config. Path provided: {str_helmizer_config_path}")
        exit(1)

    try:
        validate_helmizer_config_version(config["helmizer"]["version"].get(str))
    except KeyError:
        logging.debug("Unable to validate version")

    return config


def main():
    arguments = init_arg_parser()  # parse args
    helmizer_config = init_helmizer_config(arguments)  # parse config file
    if not arguments.skip_commands:  # run subprocesses
        run_subprocess(helmizer_config, arguments)
    kustomization = Kustomization(helmizer_config, arguments)  # render kustomization
    kustomization.render_template(arguments)  # instantiate kustomization YAML


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt or SystemExit:
        exit(1)
