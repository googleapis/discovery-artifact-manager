package php

import "text/template"

var mainTemplate = template.Must(template.New("check").Parse(`<?php
require_once './vendor/autoload.php';

include './parsedlib.php';

function isValidObject($param, $types)
{
	foreach ($types as $validType) {
		if ($param instanceof $validType) {
			return True;
		}
	}
	return False;
}

// For non-ADC based APIs, we use getClient to retrieve an initialized client
// with auth configured. This stub is provided here so compilecheck has a
// reference to the stub.
function getClient() {
  return new Google_Client();
}

{{range $sample := $}}
function {{$sample.MethodSignature.Identifier}}($parsedLib)
{
	// Initialization {{range $line := $sample.InitLines}}
	{{$line}}
	{{end}}
	// Type check
	$className = get_class(${{$sample.MethodSignature.Path}});
	$reflection = new ReflectionMethod($className, '{{$sample.MethodSignature.Method}}');

	$requiredArgs = array();
	$providedArgs = array();

	foreach ($reflection->getParameters() as $arg) {
		if (!$arg->isOptional()) {
			array_push($requiredArgs, $arg->name);
		}
	}
	{{range $param := $sample.MethodSignature.Params}}
	array_push($providedArgs, '{{$param}}');
	{{end}}
	if ((!empty($requiredArgs) || !empty($providedArgs)) && $requiredArgs !== $providedArgs) {
		throw new Exception('Incorrect parameters provided for method'
			.' {{$sample.MethodSignature.Method}} of '.$className."\n Provided:\n"
			.json_encode($providedArgs)
			."Required:\n"
			.json_encode($requiredArgs));
	}
	{{range $param := $sample.MethodSignature.Params}}
	$key = new ParameterPath($className, '{{$sample.MethodSignature.Method}}', '${{$param}}');
	if (gettype(${{$param}}) != 'object' &&
		!in_array(gettype(${{$param}}), $parsedLib[(string)$key])) {
		throw new Exception('Incorrect type of ${{$param}} in method '
			.'{{$sample.MethodSignature.Method}} of '.$className."\n"
			."Expected:\n".join('|', $parsedLib[(string)$key])."\nActual:\n".gettype(${{$param}})."\n");
	}

	if (gettype(${{$param}}) == 'object' && !isValidObject(${{$param}}, $parsedLib[(string)$key])) {
		throw new Exception('Incorrect type of ${{$param}} in method '
			.'{{$sample.MethodSignature.Method}} of '.$className."\n");
	}

	{{end}}
}

{{$sample.MethodSignature.Identifier}}($parsedLib);
{{end}}`))

var parsedLibTemplate = template.Must(template.New("parsedLib").Parse(`<?php

class ParameterPath {
	public $className;
	public $methodName;
	public $paramName;

	function __construct($className, $methodName, $paramName) {
		$this->className = $className;
		$this->methodName = $methodName;
		$this->paramName = $paramName;
	}

	function __toString() {
		return $this->className.'.'.$this->methodName.'.'.$this->paramName;
	}
}

$parsedLib = [];
{{range $pair := $}}
$key = new ParameterPath('{{$pair.Path.ClassName}}', '{{$pair.Path.MethodName}}', '{{$pair.Path.ParameterName}}');
$parsedLib[(string)$key] = array();
{{range $type := $pair.Types}}
array_push($parsedLib[(string)$key], '{{$type}}');
{{end}}
{{end}}`))
