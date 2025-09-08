#include <linux/module.h>
#include <linux/kernel.h>
#include <linux/string.h> 
#include <linux/init.h>
#include <linux/proc_fs.h> 
#include <linux/seq_file.h> 
#include <linux/mm.h> 
#include <linux/sched.h> 
#include <linux/timer.h> 
#include <linux/jiffies.h> 
#include <linux/uaccess.h>
#include <linux/tty.h>
#include <linux/sched/signal.h>
#include <linux/fs.h>        
#include <linux/slab.h>      
#include <linux/sched/mm.h>
#include <linux/binfmts.h>
#include <linux/timekeeping.h>
#include <linux/cgroup.h> // Cabecera necesaria para funciones de cgroups
#include <linux/sched/task.h> // MODIFICACIÓN: Cabecera para task_css()
#include <linux/pid_namespace.h>
#include <linux/nsproxy.h>

MODULE_LICENSE("GPL");
MODULE_AUTHOR("Abdiel Otzoy - 202300350 (Modificado por Gemini)");
MODULE_DESCRIPTION("Modulo para leer informacion de memoria y CPU de contenedores Docker en JSON");
MODULE_VERSION("1.2");

#define PROC_NAME "continfo_so1_202300350"
#define MAX_CMDLINE_LENGTH 256
#define CONTAINER_ID_LENGTH 64

// (Las funciones get_process_cmdline y get_task_state no cambian y pueden permanecer igual)
// ...
static char *get_process_cmdline(struct task_struct *task) {
    struct mm_struct *mm;
    char *cmdline, *p;
    unsigned long arg_start, arg_end, env_start;
    int i, len;

    cmdline = kmalloc(MAX_CMDLINE_LENGTH, GFP_KERNEL);
    if (!cmdline)
        return NULL;

    mm = get_task_mm(task);
    if (!mm) {
        kfree(cmdline);
        return NULL;
    }

    down_read(&mm->mmap_lock);
    arg_start = mm->arg_start;
    arg_end = mm->arg_end;
    env_start = mm->env_start;
    up_read(&mm->mmap_lock);

    len = arg_end - arg_start;

    if (len > MAX_CMDLINE_LENGTH - 1)
        len = MAX_CMDLINE_LENGTH - 1;

    if (access_process_vm(task, arg_start, cmdline, len, 0) != len) {
        mmput(mm);
        kfree(cmdline);
        return NULL;
    }

    cmdline[len] = '\0';

    p = cmdline;
    for (i = 0; i < len; i++)
        if (p[i] == '\0')
            p[i] = ' ';

    mmput(mm);
    return cmdline;
}

static const char *get_task_state(struct task_struct *task) {
    char state = task_state_to_char(task);
    switch (state) {
        case 'R': return "RUNNING";
        case 'S': return "SLEEPING";
        case 'D': return "WAITING";
        case 'T': return "STOPPED";
        case 't': return "TRACING_STOP";
        case 'Z': return "ZOMBIE";
        case 'X': return "DEAD";
        default:  return "UNKNOWN";
    }
}

/*
 * Función para mostrar la información en el archivo /proc en formato JSON
 */
static int sysinfo_show(struct seq_file *m, void *v) {
    struct sysinfo si;
    struct task_struct *task;
    unsigned long total_jiffies = jiffies;
    int first_process = 1;
    int proc_count = 0;

    // Obtenemos la información de memoria
    si_meminfo(&si);

    // Contamos los procesos activos de contenedores
    for_each_process(task) {
        // MODIFICACIÓN: Se renombró la variable para evitar conflicto con la función
        char cgroup_path_buf[256] = {0};
        
        rcu_read_lock();
        // MODIFICACIÓN: Se usa el nuevo nombre de variable en la llamada a la función
        cgroup_path(task_cgroup(task, 0), cgroup_path_buf, sizeof(cgroup_path_buf));
        rcu_read_unlock();
        
        // MODIFICACIÓN: Se usa el nuevo nombre de variable en la comparación
        if (strstr(cgroup_path_buf, "docker")) {
            proc_count++;
        }
    }

    seq_printf(m, "{\n");
    seq_printf(m, "  \"Totalram\": %lu,\n", si.totalram << (PAGE_SHIFT - 10));
    seq_printf(m, "  \"Freeram\": %lu,\n", si.freeram << (PAGE_SHIFT - 10));
    seq_printf(m, "  \"Procs\": %d,\n", proc_count);
    seq_printf(m, "  \"Processes\": [\n");

    // Iteramos sobre todos los procesos
    for_each_process(task) {
        // MODIFICACIÓN: Se renombró la variable para evitar conflicto con la función
        char cgroup_path_buf[256] = {0};

        rcu_read_lock();
        // MODIFICACIÓN: Se usa el nuevo nombre de variable en la llamada a la función
        cgroup_path(task_cgroup(task, 0), cgroup_path_buf, sizeof(cgroup_path_buf));
        rcu_read_unlock();

        // Filtramos para procesar solo si es un contenedor de Docker
        if (!strstr(cgroup_path_buf, "docker")) {
            continue; // Si no es de Docker, saltamos al siguiente proceso
        }

        // Verificamos que sea el proceso PID 1 dentro de su namespace
        if (task_pid_nr_ns(task, task->nsproxy->pid_ns_for_children) != 1) {
            continue; // No es el proceso principal del contenedor
        }

        unsigned long vsz = 0;
        unsigned long rss = 0;
        unsigned long totalram = si.totalram * (PAGE_SIZE / 1024);
        unsigned long mem_usage = 0;
        unsigned long cpu_usage = 0;
        char *cmdline = NULL;

        if (task->mm) {
            vsz = task->mm->total_vm << (PAGE_SHIFT - 10);
            rss = get_mm_rss(task->mm) << (PAGE_SHIFT - 10);
            if (totalram > 0) {
                mem_usage = (rss * 100) / totalram;
            }
        }

        unsigned long total_time = task->utime + task->stime;
        if (total_jiffies > 0) {
             cpu_usage = (total_time * 10000) / total_jiffies;
        }
       
        cmdline = get_process_cmdline(task);

        if (!first_process) {
            seq_printf(m, ",\n");
        } else {
            first_process = 0;
        }

        seq_printf(m, "     {\n");
        seq_printf(m, "        \"PID\": %d,\n", task->pid);
        seq_printf(m, "        \"Name\": \"%s\",\n", task->comm);
        seq_printf(m, "        \"Cmdline\": \"%s\",\n", cmdline ? cmdline : "N/A");
        seq_printf(m, "        \"vsz\": %lu,\n", vsz);
        seq_printf(m, "        \"rss\": %lu,\n", rss);
        seq_printf(m, "        \"Memory_Usage\": %lu.%lu,\n", mem_usage / 10, mem_usage % 10);
        seq_printf(m, "        \"CPU_Usage\": %lu.%lu,\n", cpu_usage / 100, cpu_usage % 100);
        seq_printf(m, "        \"State\": \"%s\"\n", get_task_state(task));
        seq_printf(m, "     }");

        if (cmdline) {
            kfree(cmdline);
        }
    }

    seq_printf(m, "\n   ]\n}\n");
    return 0;
}


// (Las funciones sysinfo_open, sysinfo_ops, sysinfo_init y sysinfo_exit no cambian)
// ...
static int sysinfo_open(struct inode *inode, struct file *file) {
    return single_open(file, sysinfo_show, NULL);
}

static const struct proc_ops sysinfo_ops = {
    .proc_open = sysinfo_open,
    .proc_read = seq_read,
};

static int __init sysinfo_init(void) {
    proc_create(PROC_NAME, 0, NULL, &sysinfo_ops);
    printk(KERN_INFO "sysinfo_json modulo cargado (Filtro Docker activado)\n");
    return 0;
}

static void __exit sysinfo_exit(void) {
    remove_proc_entry(PROC_NAME, NULL);
    printk(KERN_INFO "sysinfo_json modulo desinstalado\n");
}

module_init(sysinfo_init);
module_exit(sysinfo_exit);