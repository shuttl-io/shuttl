#!/usr/bin/env python3
"""
Python test application for integration testing

This app creates a simple agent with a toolkit and tool,
then starts the StdInServer to accept IPC commands.

Note: JSII interfaces must be implemented using the @jsii.implements
decorator, not by inheriting from the interface directly.
See: https://aws.github.io/jsii/user-guides/lib-user/language-specific/python/
"""

import sys
import os
import jsii

# Add the built Python package to the path
# The package will be installed in the virtual environment
from shuttl.model import (
    App,
    Agent, 
    Model,
    Toolkit,
    StdInServer,
    Secret,
)
from shuttl.model.tools import ITool, ToolArg


@jsii.implements(ITool)
class EchoTool:
    """A simple test tool that echoes its input"""
    
    def __init__(self):
        self._name = "echo"
        self._description = "Echoes the input message back"
    
    @property
    def name(self) -> str:
        return self._name
    
    @name.setter
    def name(self, value: str):
        self._name = value
    
    @property
    def description(self) -> str:
        return self._description
    
    @description.setter
    def description(self, value: str):
        self._description = value
    
    def execute(self, args):
        message = args.get("message", "no message")
        import time
        return {
            "echoed": message,
            "timestamp": int(time.time() * 1000),
        }
    
    def produce_args(self):
        return {
            "message": ToolArg(
                name="message",
                arg_type="string",
                description="The message to echo",
                required=True,
                default_value=None,
            ),
        }


@jsii.implements(ITool)
class MathTool:
    """A tool that performs simple math"""
    
    def __init__(self):
        self._name = "add"
        self._description = "Adds two numbers together"
    
    @property
    def name(self) -> str:
        return self._name
    
    @name.setter
    def name(self, value: str):
        self._name = value
    
    @property
    def description(self) -> str:
        return self._description
    
    @description.setter
    def description(self, value: str):
        self._description = value
    
    def execute(self, args):
        a = args.get("a", 0)
        b = args.get("b", 0)
        return {
            "result": a + b,
            "operation": "add",
        }
    
    def produce_args(self):
        return {
            "a": ToolArg(
                name="a",
                arg_type="number",
                description="First number",
                required=True,
                default_value=None,
            ),
            "b": ToolArg(
                name="b",
                arg_type="number",
                description="Second number",
                required=True,
                default_value=None,
            ),
        }


def main():
    # Create server
    server = StdInServer()
    
    # Create app
    app = App("PythonTestApp", server)
    
    # Create model
    model = Model(
        identifier="test-model",
        key=Secret.fromEnv("OPENAI_API_KEY"),
    )
    
    # Create toolkit with tools
    util_toolkit = Toolkit(
        name="UtilityToolkit",
        description="A toolkit with utility functions",
        tools=[EchoTool(), MathTool()],
    )
    
    # Create agent
    agent = Agent(
        name="TestAgent",
        system_prompt="You are a helpful test agent for integration testing.",
        model=model,
        toolkits=[util_toolkit],
    )
    
    # Add agent to app
    app.add_agent(agent)
    
    # Start serving
    app.serve()


if __name__ == "__main__":
    main()






















