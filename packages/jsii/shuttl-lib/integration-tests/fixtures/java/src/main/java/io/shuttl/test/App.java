package io.shuttl.test;

import io.shuttl.module.shuttl.*;
import io.shuttl.module.shuttl.tools.*;

import java.util.Arrays;
import java.util.HashMap;
import java.util.Map;

/**
 * Java test application for integration testing
 *
 * This app creates a simple agent with a toolkit and tool,
 * then starts the StdInServer to accept IPC commands.
 */
public class App {
    
    /**
     * A simple test tool that echoes its input
     */
    static class EchoTool implements ITool {
        private String name = "echo";
        private String description = "Echoes the input message back";
        
        @Override
        public String getName() {
            return name;
        }
        
        @Override
        public void setName(String name) {
            this.name = name;
        }
        
        @Override
        public String getDescription() {
            return description;
        }
        
        @Override
        public void setDescription(String description) {
            this.description = description;
        }
        
        @Override
        public Object execute(Map<String, Object> args) {
            String message = (String) args.getOrDefault("message", "no message");
            Map<String, Object> result = new HashMap<>();
            result.put("echoed", message);
            result.put("timestamp", System.currentTimeMillis());
            return result;
        }
        
        @Override
        public Map<String, ToolArg> produceArgs() {
            Map<String, ToolArg> args = new HashMap<>();
            args.put("message", ToolArg.builder()
                .name("message")
                .argType("string")
                .description("The message to echo")
                .required(true)
                .defaultValue(null)
                .build());
            return args;
        }
    }
    
    /**
     * A tool that performs simple math
     */
    static class MathTool implements ITool {
        private String name = "add";
        private String description = "Adds two numbers together";
        
        @Override
        public String getName() {
            return name;
        }
        
        @Override
        public void setName(String name) {
            this.name = name;
        }
        
        @Override
        public String getDescription() {
            return description;
        }
        
        @Override
        public void setDescription(String description) {
            this.description = description;
        }
        
        @Override
        public Object execute(Map<String, Object> args) {
            Number a = (Number) args.getOrDefault("a", 0);
            Number b = (Number) args.getOrDefault("b", 0);
            Map<String, Object> result = new HashMap<>();
            result.put("result", a.doubleValue() + b.doubleValue());
            result.put("operation", "add");
            return result;
        }
        
        @Override
        public Map<String, ToolArg> produceArgs() {
            Map<String, ToolArg> args = new HashMap<>();
            args.put("a", ToolArg.builder()
                .name("a")
                .argType("number")
                .description("First number")
                .required(true)
                .defaultValue(null)
                .build());
            args.put("b", ToolArg.builder()
                .name("b")
                .argType("number")
                .description("Second number")
                .required(true)
                .defaultValue(null)
                .build());
            return args;
        }
    }
    
    public static void main(String[] args) {
        // Create server
        StdInServer server = new StdInServer();
        
        // Create app
        io.shuttl.module.shuttl.App app = new io.shuttl.module.shuttl.App("JavaTestApp", server);
        
        // Create model
        Model model = Model.builder()
            .identifier("test-model")
            .key("test-key-12345")
            .build();
        
        // Create toolkit with tools
        Toolkit utilToolkit = Toolkit.builder()
            .name("UtilityToolkit")
            .description("A toolkit with utility functions")
            .tools(Arrays.asList(new EchoTool(), new MathTool()))
            .build();
        
        // Create agent
        Agent agent = Agent.builder()
            .name("TestAgent")
            .systemPrompt("You are a helpful test agent for integration testing.")
            .model(model)
            .toolkits(Arrays.asList(utilToolkit))
            .build();
        
        // Add agent to app
        app.addAgent(agent);
        
        // Start serving
        app.serve();
    }
}











