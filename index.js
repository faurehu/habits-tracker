child_process = require('child_process');

exports.handler = function(event, context) {
  var proc = child_process.spawn('./main', [ JSON.stringify(event) ], { stdio: 'pipe' });

  proc.stdout.on('data', function(data) {
    console.log('STDOUT: ', data.toString('utf8'))
  })

  proc.stderr.on('data', function(data) {
    console.log('STDERR: ', data.toString('utf8'))
  })

  proc.on('close', function(code) {
    if(code !== 0) {
      return context.done(new Error("Process exited with non-zero status code"));
    }

    context.done(null);
  });
}
