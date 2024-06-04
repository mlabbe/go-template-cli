#!/usr/bin/env python

import os
import sys
import shutil
import subprocess

from os.path import join as path_join

def is_host_windows():
    return os.name == 'nt'

def exe(name):
    if is_host_windows():
        return name + '.exe'
    else:
        return name

def fatal(msg):
    print(msg, file=sys.stderr)
    sys.exit(1)

def message(msg):
    if not find_arg_0param('--quiet'):
        print(msg)

def shell(cmd):
    if not find_arg_0param('--quiet'):
        print(' '.join(cmd))
    cp = subprocess.run(cmd)

    if cp.returncode != 0:
        fatal("%s failed" % ' '.join(cmd))

def get_installed_executable_path(exe):
    gopath = path_join(os.environ.get('GOPATH'), 'bin')
    if not os.environ.get('GOBIN') is None:
        gopath = os.environ.get('GOBIN')

    return path_join(gopath, exe)

def shell_backtick(cmd, shell):
    return subprocess.run(cmd, capture_output=True, shell=shell).stdout

def get_host_os_tools_bin():
    if is_host_windows():
        target = "win64"
    else:
        uname_os = shell_backtick('uname -s', shell=True).lower().decode("utf-8").strip()
        machine = shell_backtick('uname -m', shell=True).lower().decode("utf-8").strip()
        target = "%s-%s" % (uname_os, machine)

    return path_join(os.getenv('FTG_PROJECT_ROOT'), 'ftg-tools-bin', target, 'bin')

def find_arg_0param(expected_arg):
    for arg in sys.argv:
        if arg == expected_arg:
            return True

    return False


#
# main 
#

EXE=exe("tpl")
os.chdir(path_join('cmd', 'tpl'))
shell(['go', 'install'])

# internal only
if os.environ.get('FTG_PROJECT_ROOT') != None and not find_arg_0param('--skip-install'):
    src_path = get_installed_executable_path(EXE)
    dst_path = path_join(get_host_os_tools_bin(), EXE)
    shutil.copy2(src_path, dst_path)
    message("%s installed to '%s'" % (EXE, dst_path))
