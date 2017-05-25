#!/usr/bin/env python

from setuptools import find_packages
from setuptools import setup

setup(name='mockgen',
      version='1.0',
      packages=find_packages(),
      install_requires=['rstr'],
      entry_points={
          'console_scripts': [
              ('generate_mock_discovery_document = '
               'mockgen.generate_mock_discovery_document:main'),
              'generate_mock_server = mockgen.generate_mock_server:main',
              ('generate_mock_value_override = '
               'mockgen.generate_mock_value_override:main')
          ],
      })

