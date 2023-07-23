#!/usr/bin/env python3
"""
Helmizer - Generates `kustomization.yaml` for your locally-rendered YAML manifests,
such as from Helm templates or plain YAML manifests
"""
import argparse
import json
import logging
import subprocess
from os import path
from os import walk
from sys import exit
from sys import stdout

import confuse
import yaml
from confuse.exceptions import NotFoundError
from validators import url as validate_url


class Kustomization:
    def __init__(self, helmizer_config, arguments) -> None:
        self.helmizer_config = helmizer_config
        self.arguments = arguments
        self.yaml: dict = dict()

        # get apiVersion
        self.yaml["apiVersion"] = self.get_api_version()

        # get kind
        self.yaml["kind"] = "Kustomization"

        # get strings
        for list_of_keys in ["namePrefix", "nameSuffix", "namespace"]:
            try:
                key_string: str = self.get_str(list_of_keys)
                if key_string:
                    self.yaml[list_of_keys] = key_string
            except NotFoundError:
                pass

        # get lists
        for list_of_keys in [
            "buildMetadata",
            "configMapGenerator",
            "images",
            "patches",
            "patchesJson6902",
            "replacements",
            "replicas",
            "secretGenerator",
            "vars",
        ]:
            try:
                key_list = self.get_list(list_of_keys)
                if key_list:
                    self.yaml[list_of_keys] = key_list
            except NotFoundError:
                pass

        # get dicts
        for list_of_keys in ["commonAnnotations", "commonLabels", "generatorOptions", "openapi", "sortOptions"]:
            try:
                key_dict = self.get_dict(list_of_keys)
                if key_dict:
                    self.yaml[list_of_keys] = key_dict
            except NotFoundError:
                pass

        # get files
        for list_of_keys in ["crds", "components", "patchesStrategicMerge", "resources"]:
            try:
                key_paths = self.get_files(arguments, list_of_keys)
                if key_paths:
                    self.yaml[list_of_keys] = key_paths
            except NotFoundError:
                pass

    def sort_keys(self, arguments) -> bool:
        if arguments.no_sort_keys:
            return True
        else:
            try:
                sort_keys: bool = self.helmizer_config["helmizer"]["sort-keys"].get(bool)
                if sort_keys:
                    for key in ["crds", "components", "patchesStrategicMerge", "resources"]:
                        try:
                            self.yaml[key].sort()
                        except KeyError:
                            pass
                    logging.debug("keys sorted")
            except NotFoundError:
                for key in ["crds", "components", "patchesStrategicMerge", "resources"]:
                    try:
                        self.yaml[key].sort()
                    except KeyError:
                        pass
            finally:
                return True

    def print_kustomization(self) -> bool:
        try:
            print(yaml.dump(self.yaml, sort_keys=False))
        except KeyError:
            print(yaml.dump(self.yaml, sort_keys=False))
        finally:
            return True

    def write_kustomization(self, arguments) -> bool:
        try:
            if arguments.dry_run:
                logging.debug("Performing dry-run, not writing to a file system")
                return True
            elif self.helmizer_config["helmizer"]["dry-run"].get(bool):
                logging.debug("Performing dry-run, not writing to a file system")
                return True
        except NotFoundError:
            pass

        # identify kustomization file's parent directory
        kustomization_directory: str = path.dirname(path.abspath(path.normpath(arguments.helmizer_config)))

        # identify kustomization file name
        kustomization_file_name: str = str()
        try:
            kustomization_file_name = self.helmizer_config["helmizer"]["kustomization-file-name"].get(str)
        except NotFoundError:
            kustomization_file_name = "kustomization.yaml"

        # write to file
        try:
            kustomization_file_path: str = path.normpath(f"{kustomization_directory}/{kustomization_file_name}")
            with open(kustomization_file_path, "w") as file:
                file.write(yaml.dump(self.yaml))
                logging.debug(f"Successfully wrote to file: {path.abspath(kustomization_file_path)}")
            return True
        except IsADirectoryError as e:
            raise e
        except TypeError as e:
            raise e

    def render_template(self, arguments) -> bool:
        logging.debug("Rendering template")
        self.sort_keys(arguments)
        self.print_kustomization()
        self.write_kustomization(arguments)
        return True

    def get_api_version(self) -> str:
        api_version: str = str()
        try:
            api_version = self.helmizer_config["kustomize"]["apiVersion"].get(str)
        except NotFoundError:
            api_version = "kustomize.config.k8s.io/v1beta1"
        finally:
            logging.debug(f"apiVersion: {api_version}")
            return api_version

    def get_files(self, arguments, key: str) -> list:
        target_paths: list = list()
        final_target_paths: list = list()
        kustomization_path: str = str()

        try:
            # test if the key to configure is even defined in input helmizer config
            kustomization_children: list = self.helmizer_config["kustomize"][key].get(list)
            kustomization_directory: str = str()
            try:
                kustomization_directory = self.helmizer_config["helmizer"]["kustomization-directory"].get(str)
            except NotFoundError:
                kustomization_directory = "."

            kustomization_path = path.dirname(
                path.abspath(
                    path.normpath(
                        path.join(
                            arguments.helmizer_config,
                            kustomization_directory,
                        )
                    )
                )
            )

            if len(kustomization_children) > 0:
                # each target path
                for target_path in kustomization_children:
                    child_path: str = path.abspath(path.join(kustomization_path, target_path))

                    # walk directory
                    if path.isdir(child_path):
                        for dirpath, _, filenames in walk(child_path):
                            for filename in filenames:
                                target_paths.append(path.join(dirpath, filename))

                    # file
                    elif path.isfile(child_path):
                        target_paths.append(child_path)

                    # url
                    elif validate_url(child_path):
                        target_paths.append(child_path)

                # remove any ignored files
                try:
                    # walk directory to remove multiple files
                    for ignore in self.helmizer_config["helmizer"]["ignore"].get(list):
                        ignore_abspath: str = path.abspath(path.join(kustomization_path, ignore))
                        if path.isdir(ignore_abspath):
                            for dirpath, _, filenames in walk(ignore_abspath):
                                for filename in filenames:
                                    file_path: str = path.join(dirpath, filename)
                                    logging.debug(f"Removing ignored file from final list: {file_path}")
                                    target_paths.remove(file_path)
                        # remove a file
                        else:
                            logging.debug(f"Removing ignored file from final list: {path.join(kustomization_path, ignore)}")
                            target_paths.remove(path.join(kustomization_path, ignore))  # just one file
                except ValueError:
                    pass
                except NotFoundError:
                    pass

                # convert absolute paths into paths relative to the kustomization directory
                for final_target_path in target_paths:
                    final_target_paths.append(path.relpath(final_target_path, kustomization_path))

                return final_target_paths

        except NotFoundError:
            logging.debug(f"key not found: {key}")
        except KeyError:
            logging.debug(f"key not found: {key}")
        except TypeError:
            logging.debug(f"key not of valid type: {key}")
        finally:
            return final_target_paths

    def get_dict(self, key: str) -> dict:
        raw_yaml: dict = dict()
        try:
            if self.helmizer_config["kustomize"][key].get(dict):
                raw_yaml = json.loads(json.dumps(self.helmizer_config["kustomize"][key].get(dict)))
                logging.debug(f"{key}: {raw_yaml}")
        except NotFoundError:
            logging.debug(f"key not found: {key}")
            return raw_yaml
        except KeyError:
            logging.debug(f"key not found: {key}")
            return raw_yaml
        except TypeError:
            logging.debug(f"key not of valid type: {key}")
            return raw_yaml
        return raw_yaml

    def get_str(self, key: str) -> str:
        raw_yaml: str = str()
        try:
            if self.helmizer_config["kustomize"][key].get(str):
                raw_yaml = json.loads(json.dumps(self.helmizer_config["kustomize"][key].get(str)))
                logging.debug(f"{key}: {raw_yaml}")
        except NotFoundError:
            logging.debug(f"key not found: {key}")
        except KeyError:
            logging.debug(f"key not found: {key}")
        except TypeError:
            pass
        return raw_yaml

    def get_list(self, key: str) -> list:
        raw_yaml: list = list()
        try:
            if len(self.helmizer_config["kustomize"][key].get(list)) > 0:
                raw_yaml = json.loads(json.dumps(self.helmizer_config["kustomize"][key].get(list)))
                logging.debug(f"{key}: {raw_yaml}")
        except NotFoundError:
            logging.debug(f"key not found: {key}")
        except KeyError:
            logging.debug(f"key not found: {key}")
        except TypeError:
            logging.debug(f"key not of valid type: {key}")
        return raw_yaml


def run_subprocess(helmizer_config, arguments):
    # TODO return type checking
    subprocess_working_directory = path.dirname(path.abspath(path.normpath(arguments.helmizer_config)))
    logging.debug(f"Subprocess working directory: {subprocess_working_directory}")
    command_string: list = list()
    try:
        for config_command in helmizer_config["helmizer"]["commandSequence"]:
            # construct command(s)
            if config_command["command"]:
                command: str = config_command["command"].get(str)
                if config_command["args"]:
                    args: str = " ".join(config_command["args"].get(list))  # combine list elements into space-delimited
                    command_string.append(f"{command} {args}")
                else:
                    command_string.append(f"{command}")

        # execute
        for command in command_string:
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
            "--no-sort-keys",
            dest="no_sort_keys",
            action="store_true",
            help="disables alphabetical sorting of sub-keys (such as 'resources') in output kustomization file",
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
        args.add_argument("--version", "-v", action="version", version="v0.13.0")
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


def validate_helmizer_config_version(helmizer_config_version: str) -> bool:
    # TODO actually validate it
    logging.debug(f"validating helmizer config version: {helmizer_config_version}")
    return True


def init_helmizer_config(arguments) -> object:
    helmizer_config_path = arguments.helmizer_config

    config = confuse.Configuration("helmizer", __name__)
    try:
        if helmizer_config_path:
            logging.debug(f"Trying helmizer config path from argument: {helmizer_config_path}")
            config.set_file(path.normpath(helmizer_config_path))
            logging.debug(f"parsed config: {config}")

    # no config file found. Give up
    except confuse.exceptions.ConfigReadError:
        logging.error(f"Unable to locate helmizer config. Path provided: {helmizer_config_path}")
        exit(1)

    try:
        validate_helmizer_config_version(config["helmizer"]["version"].get(str))
    except NotFoundError:
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
    except SystemExit:
        exit(1)
    except KeyboardInterrupt:
        exit(1)
